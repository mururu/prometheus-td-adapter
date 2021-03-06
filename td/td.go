package td

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/common/model"
	"github.com/ugorji/go/codec"

	td_client "github.com/treasure-data/td-client-go"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

type Config struct {
	apiKey string
	db     string
	table  string
	router td_client.EndpointRouter // for test
	port   int                      // for test
}

func ParseFlags(cfg *Config) *Config {
	flag.StringVar(&cfg.apiKey, "td.apikey", "", "The API Key for Treasure Data")
	flag.StringVar(&cfg.db, "td.db", "", "The Database Name for Treasure Data")
	flag.StringVar(&cfg.table, "td.table", "", "The Table Name for Treasure Data")

	// Enable set config via enviroment variables
	flag.VisitAll(func(f *flag.Flag) {
		keys := []string{"td.apikey", "td.db", "td.table"}
		for _, key := range keys {
			if f.Name == key {
				env := strings.Replace(strings.ToUpper(key), ".", "_", -1)
				if s := os.Getenv(env); s != "" {
					f.Value.Set(s)
				}
			}
		}
	})
	return cfg
}

func validateConfig(cfg *Config) error {
	if cfg.apiKey == "" {
		return fmt.Errorf("td.apikey is required")
	}
	if cfg.db == "" {
		return fmt.Errorf("td.db is required")
	}
	if cfg.table == "" {
		return fmt.Errorf("td.table is required")
	}

	return nil
}

type Client struct {
	logger log.Logger

	client *td_client.TDClient
	db     string
	table  string
}

func checkTableExistence(c *td_client.TDClient, db string, table string) error {
	retval := bytes.Buffer{}
	gzip.NewWriter(&retval).Close()
	empty := retval.Bytes()

	_, err := c.Import(db, table, "msgpack.gz", (td_client.InMemoryBlob)(empty), "")
	if apierr, ok := err.(*td_client.APIError); ok {
		if apierr.Type == td_client.NotFoundError {
			return fmt.Errorf("Table %s.%s does not exist on Treasure Data", db, table)
		}
	}
	return err
}

func NewClient(logger log.Logger, cfg *Config) *Client {
	if logger == nil {
		logger = log.NewNopLogger()
	}

	if err := validateConfig(cfg); err != nil {
		level.Error(logger).Log("msg", "Failed to parse td options", "err", err)
		os.Exit(1)
	}

	settings := td_client.Settings{
		ApiKey: cfg.apiKey,
	}
	if cfg.router != nil {
		settings.Router = cfg.router
	}
	if cfg.port != 0 {
		settings.Port = cfg.port
	}

	c, err := td_client.NewTDClient(settings)
	if err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}

	if err = checkTableExistence(c, cfg.db, cfg.table); err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}

	// TODO: validate db, table
	return &Client{
		logger: logger,
		client: c,
		db:     cfg.db,
		table:  cfg.table,
	}
}

func compressWithGzip(b []byte) []byte {
	retval := bytes.Buffer{}
	w := gzip.NewWriter(&retval)
	w.Write(b)
	w.Close()
	return retval.Bytes()
}

func generateUniqueId() string {
	now := time.Now().UTC()
	sec := now.Second()
	usec := now.UnixNano() / int64(time.Microsecond)
	u1 := ((sec*1000*1000 + int(usec)) << 12) | rand.Intn(0xfff)
	a := []uint32{uint32(u1) >> 32, uint32(u1) & 0xffffffff, uint32(rand.Intn(0xffffffff)), uint32(rand.Intn(0xffffffff))}

	var buffer bytes.Buffer
	for _, u := range a {
		buffer.WriteString(fmt.Sprintf("%02s", strconv.FormatUint(uint64(byte(u>>24)), 16)))
		buffer.WriteString(fmt.Sprintf("%02s", strconv.FormatUint(uint64(byte(u>>16)), 16)))
		buffer.WriteString(fmt.Sprintf("%02s", strconv.FormatUint(uint64(byte(u>>8)), 16)))
		buffer.WriteString(fmt.Sprintf("%02s", strconv.FormatUint(uint64(byte(u)), 16)))
	}
	return buffer.String()
}

func (c *Client) Write(samples model.Samples) error {
	data := bytes.Buffer{}
	handle := codec.MsgpackHandle{}
	encoder := codec.NewEncoder(&data, &handle)
	for _, s := range samples {
		record := map[string]interface{}{
			"time":  s.Timestamp.Unix(),
			"name":  string(s.Metric[model.MetricNameLabel]),
			"value": float64(s.Value),
		}
		for l, v := range s.Metric {
			if l != model.MetricNameLabel {
				record["label_"+string(l)] = string(v)
			}
		}

		encoder.Encode(record)
	}

	payload := compressWithGzip(data.Bytes())
	uniqueId := generateUniqueId()
	_, err := c.client.Import(c.db, c.table, "msgpack.gz", (td_client.InMemoryBlob)(payload), uniqueId)
	if err != nil {
		return err
	}

	level.Debug(c.logger).Log("msg", "wrote sample in td", "num_sumples", len(samples), "payload_size", len(payload))

	return nil
}

// Name identifies the client as an Treasure Data client.
func (c Client) Name() string {
	return "TreasureData"
}
