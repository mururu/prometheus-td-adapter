# prometheus-td-adapter

This allow Prometheus to use Treasure Data as long-term storage.

## Build

[dep](https://github.com/golang/dep) is required for dependency management.

```
$ make deps
$ make
```

## Usage

```
$ ./prometheus-td-adapter -td.apikey=yourapikey -td.db=yourdb -td.table=yourtable
```

You can pass the td related parameters by environment variables: `TD_APIKEY`, `TD_DB` and `TD_TABLE`.

For other options, see ` ./prometheus-td-adapter -h`.

## Prometheus Configuration

```
remote_write:
  - url: "http://localhost:9201/write"
```

## Data Model

Metrics is stored in Treasure Data as below.

- `time`: timestamp
- `value`: metric value
- `name`: metric name
- `label_*`: labels (automatically prefixed by "label_")

### Example

```
{
  "time": 1508050569,
  "value": 802713,
  "name": "node_network_transmit_packets",
  "label_job": "prometheus",
  "label_instance": "localhost:9100",
  "label_device": "lo0"
}
```

## TODO

- READ
