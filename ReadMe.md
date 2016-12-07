
Uses reflection to override values in an arbitrary object based on environment variables, following the [convention outlined by Grafana/InfluxData](http://docs.grafana.org/installation/configuration/#using-environment-variables).

Ripped from [influxdb source code](
https://github.com/influxdata/influxdb/blob/77e2c80a4f220770a2da00bc1ff048c762f8cc66/cmd/influxd/run/config.go#L182) with some modifications.

See the test for a usage example.
