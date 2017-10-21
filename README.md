# prometheus-td-adapter

This allow Prometheus to use Treasure Data as long-term storage.

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