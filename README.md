# Prometheus SCW Service Discovery

prometheus-scw-sd retrieves your Scaleway server list and converts to Prometheus targets.

This project is adapted from the official [Consul SD](https://github.com/prometheus/prometheus/tree/master/documentation/examples/custom-sd).

An official Prometheus [blog post](https://prometheus.io/blog/2018/07/05/implementing-custom-sd/) explains how to write a Custom Service Discovery.

## Usage

Download the binary from the last release, corresponding to your own architecture.

Help:
```
./scw-sd -h
```

Start the discoverer:
```
./scw-sd --token="$TOKEN" --output.file="scw_sd.json"
```

Using servers private IP, custom port and time interval in second:
```
./scw-sd \
    --token="$TOKEN"            \
    --output.file="scw_sd.json" \
    --time.interval="90"        \
    --port="1234"               \
    --private
```

## Config

prometheus-scw-sd outputs a json file containing targets to scrape.
You need to include this file in your `prometheus.yml`.

```yml
scrape_configs:
  - job_name: 'scw-sd'
    file_sd_configs:
      - files:
        - path/to/scw_sd.json
```

## Labels

prometheus-scw-sd scrape Scaleway servers tags as labels, as comma separated list of strings.
This allows you to use regex substitution for relabelling.
We surround the separated list with the separator as well. This way regular expressions
in relabeling rules don't have to consider tag positions.


## Contribute

Custom SD is part of the Prometheus project, so you'll need to import Prometheus from github, and build Scaleway SD from it.

Clone Prometheus and Scaleway client dependency:
```
go get -u -v github.com/prometheus/prometheus/...
go get -u -v github.com/scaleway/go-scaleway/...
```

Move inside Custom SD folder:
```
cd $GOPATH/src/github.com/prometheus/prometheus/documentation/examples/custom-sd
```

Clone prometheus-scw-sd:
```
git clone https://github.com/scaleway/prometheus-scw-sd && cd scw-sd
```

Build:
```
go build
```
