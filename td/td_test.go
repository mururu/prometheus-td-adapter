package td

import (
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/prometheus/common/model"
	td_client "github.com/treasure-data/td-client-go"
	"github.com/ugorji/go/codec"
)

func TestClient(t *testing.T) {
	samples := model.Samples{
		{
			Metric: model.Metric{
				model.MetricNameLabel: "testmetric",
				"test_label":          "test_label_value1",
			},
			Timestamp: model.Time(123456789123),
			Value:     1.23,
		},
		{
			Metric: model.Metric{
				model.MetricNameLabel: "testmetric",
				"test_label":          "test_label_value2",
			},
			Timestamp: model.Time(123456789321),
			Value:     5.1234,
		},
		{
			Metric: model.Metric{
				model.MetricNameLabel: "nan_value",
			},
			Timestamp: model.Time(123123123123),
			Value:     model.SampleValue(math.NaN()),
		},
		{
			Metric: model.Metric{
				model.MetricNameLabel: "pos_inf_value",
			},
			Timestamp: model.Time(987654321234),
			Value:     model.SampleValue(math.Inf(1)),
		},
		{
			Metric: model.Metric{
				model.MetricNameLabel: "neg_inf_value",
			},
			Timestamp: model.Time(1234533456),
			Value:     model.SampleValue(math.Inf(-1)),
		},
	}

	apiKey := "testapikey"
	db := "testdb"
	table := "testtable"

	expectedAuthorizationHeader := "TD1 apiKey"
	expectedPath := path.Join("/v3/table/import_with_id", db, table, "[0-9a-f]{32}", "msgpack.gz")

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "PUT" {
				t.Fatalf("Unexpected method; expected POST, got %s", r.Method)
			}
			if r.Header.Get("Authorization") != "TD1 "+apiKey {
				t.Fatalf("Unexpected authorization header; expected %s, got %s", expectedAuthorizationHeader, r.Header.Get("Authorization"))
			}
			matched, err := regexp.MatchString(expectedPath, r.URL.Path)
			if err != nil {
				t.Fatalf("Error compiling path regexp; path: ", expectedPath)
			}
			if !matched {
				t.Fatalf("Unexpected path; expected %s, got %s", expectedPath, r.URL.Path)
			}

			reader, _ := gzip.NewReader(r.Body)
			b, err := ioutil.ReadAll(reader)
			if err != nil {
				t.Fatalf("Error read gzip: %v", err)
			}
			if err := reader.Close(); err != nil {
				t.Fatalf("Reader.Close: %v", err)
			}

			handle := codec.MsgpackHandle{}
			decoder := codec.NewDecoderBytes(b, &handle)
			for _, s := range samples {
				r := map[string]interface{}{}
				if err = decoder.Decode(r); err != nil {
					t.Fatalf("Error decode: %v", err)
				}
				if r["time"].(uint64) != uint64(s.Timestamp.Unix()) {
					t.Fatalf("Error time, expected: %d, got: %d", uint64(s.Timestamp.Unix()), r["time"].(uint64))
				}
				if string(r["name"].([]byte)) != string(s.Metric[model.MetricNameLabel]) {
					t.Fatalf("Error name, expected: %s, got: %s", string(s.Metric[model.MetricNameLabel]), string(r["name"].([]byte)))
				}
				for l, v := range s.Metric {
					if l != model.MetricNameLabel {
						if string(r["label_"+string(l)].([]byte)) != string(v) {
							t.Fatalf("Error label, expected", string(v), string(r["label_"+string(l)].([]byte)))
						}
					}
				}
				if math.IsNaN(float64(s.Value)) {
					if !math.IsNaN(r["value"].(float64)) {
						t.Fatalf("Error value, expected: %f, got: %f", float64(s.Value), r["value"].(float64))
					}
				} else {
					if r["value"].(float64) != float64(s.Value) {
						t.Fatalf("Error value, expected: %f, got: %f", float64(s.Value), r["value"].(float64))
					}
				}
			}

			fmt.Fprint(w, `{"unique_id": "", "database": "", "table": "", "md5_hex": "", "elapsed_time": 1.0}`)
		},
	))
	defer server.Close()

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("Error parsing url: %s", err)
	}
	splits := strings.Split(parsed.Host, ":")
	host := splits[0]
	port, err := strconv.Atoi(splits[1])
	if err != nil {
		t.Fatalf("Error parsing url: %s", err)
	}
	testRouter := &td_client.FixedEndpointRouter{
		Endpoint: host,
	}

	cfg := &Config{
		apiKey: apiKey,
		db:     db,
		table:  table,
		router: td_client.EndpointRouter(testRouter),
		port:   port,
	}

	c := NewClient(nil, cfg)

	if err := c.Write(samples); err != nil {
		t.Fatalf("Error sending samples: %s", err)
	}
}
