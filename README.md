#### Prometheus Scaleway SD

prometheus-scw-sd retreives your Scaleway server list and converts to Prometheus targets.

This project is adapted from official example [here](https://github.com/prometheus/prometheus/tree/master/documentation/examples/custom-sd)

Blog post about custom service discovery [here](https://prometheus.io/blog/2018/07/05/implementing-custom-sd/)

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
