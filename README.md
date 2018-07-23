# Prometheus Scaleway SD

prometheus-scw-sd retreives your Scaleway server list and converts to Prometheus targets.

This project is adapted from official [Consul SD](https://github.com/prometheus/prometheus/tree/master/documentation/examples/custom-sd)

An official Prometheus [blog post](https://prometheus.io/blog/2018/07/05/implementing-custom-sd/) explains how to write a Custom Service Discovery.

#### Setup

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

Clone Scaleway SD:
```
git clone https://github.com/scaleway/prometheus-scw-sd && cd scw-sd
```

Build:
```
go build
```

#### Usage

Help:
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

#### Labels

Prometheus SD scrape Scaleway servers tags as labels, as comma separated list of strings.
This allows you to use regex substitution for relabelling.
