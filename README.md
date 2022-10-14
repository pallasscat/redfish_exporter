# redfish_exporter
A Prometheus exporter for server metrics exposed by DMTF’s Redfish, using [gofish](https://github.com/stmcginnis/gofish) library.

This is a work in progress.

## Configuration and usage

Exporter expects a YAML config file with targets and authentication parameters in the following format:

```yaml
'https://redfish-server.local':
  username: 'user'
  password: 'pass'
  # do not enforce SSL certificate validity
  insecure: true
```

The exporter would be then started with:

```shell
./redfish_exporter -listen-address 0.0.0.0:10015 -config-path ./config.yml
```

The exporter follows the [multi-target exporter pattern](https://prometheus.io/docs/guides/multi-target-exporter), an example request:

```shell
curl 'localhost:10015/redfish?target=redfish-server.local'
```

## Issues / improvements

- This exporter does not have a [port allocated to it](https://github.com/prometheus/prometheus/wiki/Default-port-allocations)
- No tests
- No way to dynamically add / exclude collectors

## Tested Redfish implementations

- Dell PowerEdge servers running iDRAC8, iDRAC9
