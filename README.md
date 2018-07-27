# Prometheus SCW Service Discovery

Prometheus SCW Service Discovery converts your Scaleway server list and converts to Prometheus targets.

This project is adapted from the official [Consul SD](https://github.com/prometheus/prometheus/tree/master/documentation/examples/custom-sd).

An official Prometheus [blog post](https://prometheus.io/blog/2018/07/05/implementing-custom-sd/) explains how to write a Custom Service Discovery.

## Usage

Download the binary from the last release, corresponding to your own architecture.

Help:
```
./prometheus-scw-sd -h
```

Start the discoverer:
```
./prometheus-scw-sd --token="$TOKEN" --output.file="prometheus-scw-sd.json"
```

Using servers private IP, custom port and time interval in second:
```
./prometheus-scw-sd \
    --token="$TOKEN"                       \
    --output.file="prometheus-scw-sd.json" \
    --time.interval="90"                   \
    --port="1234"                          \
    --private
```

## Config

Prometheus SCW Service Discovery outputs a json file containing targets to scrape.
You need to include this file in your `prometheus.yml`.

```yml
scrape_configs:
  - job_name: 'scw-sd'
    file_sd_configs:
      - files:
        - path/to/prometheus-scw-sd.json
```

## Labels

Prometheus SCW Service Discovery scrapes Scaleway servers tags as labels, as comma separated list of strings.
This allows you to use regex substitution for relabelling.
We surround the separated list with the separator as well. This way regular expressions
in relabeling rules don't have to consider tag positions.


## Contribute

Custom SD is part of the Prometheus project, so you'll need to import Prometheus from github, and build Scaleway SD from it.

Clone prometheus-scw-sd:
```
git clone https://github.com/scaleway/prometheus-scw-sd
```

Get dependencies:
```
go get -u -v gopkg.in/alecthomas/kingpin.v2/...
go get -u -v github.com/go-kit/kit/log/...
go get -u -v github.com/scaleway/go-scaleway/...
```

Build:
```
go build
```
