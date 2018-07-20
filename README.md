# Prometheus Scaleway SD

prometheus-scw-sd retreives your Scaleway server list and converts to Prometheus targets.

This project is adapted from [official example](https://github.com/prometheus/prometheus/tree/master/documentation/examples/custom-sd).

An official Prometheus [blog post](https://prometheus.io/blog/2018/07/05/implementing-custom-sd/) explains how to write your custom service discovery.

Build:
```
go build
```

Usage:
```
./scw-sd -h
```

Start the discoverer:
```
./scw-sd --token="$TOKEN" --output.file="scw_sd.json"
```

Using servers private IP, custom port and time interval:
```
./scw-sd \
    --token="$TOKEN"            \
    --output.file="scw_sd.json" \
    --time.interval             \
    --port="1234"               \
    --private
```
