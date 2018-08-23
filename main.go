// Copyright 2018 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/model"
	"github.com/prometheus/common/version"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/scaleway/go-scaleway"
	"github.com/scaleway/go-scaleway/types"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	a            = kingpin.New("sd adapter usage", "Tool to generate Prometheus file_sd target files for Scaleway.")
	outputf      = a.Flag("output.file", "The output filename for file_sd compatible file.").Default("scw.json").String()
	organization = a.Flag("scw.organization", "The Scaleway organization (access key).").Required().String()
	region       = a.Flag("scw.region", "The Scaleway region.").Default("par1").String()
	token        = a.Flag("scw.token", "The authentication token (secret key).").Default("").String()
	tokenf       = a.Flag("scw.token-file", "The authentication token file.").Default("").String()
	refresh      = a.Flag("target.refresh", "The refresh interval (in seconds).").Default("30").Int()
	port         = a.Flag("target.port", "The default port number for targets.").Default("80").Int()
	listen       = a.Flag("web.listen-address", "The listen address.").Default(":9465").String()

	scwPrefix = model.MetaLabelPrefix + "scaleway_"
	// archLabel is the name for the label containing the server's architecture.
	archLabel = scwPrefix + "architecture"
	// commercialTypeLabel is the name for the label containing the server's commercial type.
	commercialTypeLabel = scwPrefix + "commercial_type"
	// identifierLabel is the name for the label containing the server's identifier.
	identifierLabel = scwPrefix + "identifier"
	// nodeLabel is the name for the label containing the server's name.
	nameLabel = scwPrefix + "name"
	// imageIDLabel is the name for the label containing the server's image ID.
	imageIDLabel = scwPrefix + "image_id"
	// imageNameLabel is the name for the label containing the server's image name.
	imageNameLabel = scwPrefix + "image_name"
	// orgLabel is the name for the label containing the server's organization.
	orgLabel = scwPrefix + "organization"
	// privateIPLabel is the name for the label containing the server's private IP.
	privateIPLabel = scwPrefix + "private_ip"
	// publicIPLabel is the name for the label containing the server's public IP.
	publicIPLabel = scwPrefix + "public_ip"
	// stateLabel is the name for the label containing the server's state.
	stateLabel = scwPrefix + "state"
	// tagsLabel is the name for the label containing all the server's tags.
	tagsLabel = scwPrefix + "tags"
	// platformLabel is the name for the label containing all the server's platform location.
	platformLabel = scwPrefix + "platform_id"
	// hypervisorLabel is the name for the label containing all the server's hypervisor location.
	hypervisorLabel = scwPrefix + "hypervisor_id"
	// nodeLabel is the name for the label containing all the server's node location.
	nodeLabel = scwPrefix + "node_id"
	// bladeLabel is the name for the label containing all the server's blade location.
	bladeLabel = scwPrefix + "blade_id"
	// chassisLabel is the name for the label containing all the server's chassis location.
	chassisLabel = scwPrefix + "chassis_id"
	// clusterLabel is the name for the label containing all the server's cluster location.
	clusterLabel = scwPrefix + "cluster_id"
	// zoneLabel is the name for the label containing all the server's zone location.
	zoneLabel = scwPrefix + "zone_id"
)

var (
	reg             = prometheus.NewRegistry()
	requestDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "prometheus_scaleway_sd_request_duration_seconds",
			Help:    "Histogram of latencies for requests to the Scaleway API.",
			Buckets: []float64{0.001, 0.01, 0.1, 0.5, 1.0, 2.0, 5.0, 10.0},
		},
	)
	requestFailures = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "prometheus_scaleway_sd_request_failures_total",
			Help: "Total number of failed requests to the Scaleway API.",
		},
	)
)

func init() {
	reg.MustRegister(prometheus.NewProcessCollector(os.Getpid(), ""))
	reg.MustRegister(prometheus.NewGoCollector())
	reg.MustRegister(version.NewCollector("prometheus_scaleway_sd"))
	reg.MustRegister(requestDuration)
	reg.MustRegister(requestFailures)
}

type scwLogger struct {
	log.Logger
}

// LogHTTP implements the Logger interface of the Scaleway API.
func (l *scwLogger) LogHTTP(r *http.Request) {
	level.Debug(l).Log("msg", "HTTP request", "method", r.Method, "url", r.URL.String())
}

// Fatalf implements the Logger interface of the Scaleway API.
func (l *scwLogger) Fatalf(format string, v ...interface{}) {
	level.Error(l).Log("msg", fmt.Sprintf(format, v...))
	os.Exit(1)
}

// Debugf implements the Logger interface of the Scaleway API.
func (l *scwLogger) Debugf(format string, v ...interface{}) {
	level.Debug(l).Log("msg", fmt.Sprintf(format, v...))
}

// Infof implements the Logger interface of the Scaleway API.
func (l *scwLogger) Infof(format string, v ...interface{}) {
	level.Info(l).Log("msg", fmt.Sprintf(format, v...))
}

// Warnf implements the Logger interface of the Scaleway API.
func (l *scwLogger) Warnf(format string, v ...interface{}) {
	level.Warn(l).Log("msg", fmt.Sprintf(format, v...))
}

// Warnf implements the Logger interface of the promhttp package.
func (l *scwLogger) Println(v ...interface{}) {
	level.Error(l).Log("msg", fmt.Sprintln(v...))
}

// scwDiscoverer retrieves target information from the Scaleway API.
type scwDiscoverer struct {
	client    *api.ScalewayAPI
	port      int
	refresh   int
	separator string
	lasts     map[string]struct{}
	logger    log.Logger
}

func (d *scwDiscoverer) createTarget(srv *types.ScalewayServer) *targetgroup.Group {
	var tags string
	if len(srv.Tags) > 0 {
		tags = d.separator + strings.Join(srv.Tags, d.separator) + d.separator
	}

	addr := net.JoinHostPort(srv.PrivateIP, fmt.Sprintf("%d", d.port))

	return &targetgroup.Group{
		Source: fmt.Sprintf("scaleway/%s", srv.Identifier),
		Targets: []model.LabelSet{
			model.LabelSet{
				model.AddressLabel: model.LabelValue(addr),
			},
		},
		Labels: model.LabelSet{
			model.AddressLabel:                   model.LabelValue(addr),
			model.LabelName(archLabel):           model.LabelValue(srv.Arch),
			model.LabelName(commercialTypeLabel): model.LabelValue(srv.CommercialType),
			model.LabelName(identifierLabel):     model.LabelValue(srv.Identifier),
			model.LabelName(imageIDLabel):        model.LabelValue(srv.Image.Identifier),
			model.LabelName(imageNameLabel):      model.LabelValue(srv.Image.Name),
			model.LabelName(nameLabel):           model.LabelValue(srv.Name),
			model.LabelName(orgLabel):            model.LabelValue(srv.Organization),
			model.LabelName(privateIPLabel):      model.LabelValue(srv.PrivateIP),
			model.LabelName(publicIPLabel):       model.LabelValue(srv.PublicAddress.IP),
			model.LabelName(stateLabel):          model.LabelValue(srv.State),
			model.LabelName(tagsLabel):           model.LabelValue(tags),
			model.LabelName(platformLabel):       model.LabelValue(srv.Location.Platform),
			model.LabelName(hypervisorLabel):     model.LabelValue(srv.Location.Hypervisor),
			model.LabelName(nodeLabel):           model.LabelValue(srv.Location.Node),
			model.LabelName(bladeLabel):          model.LabelValue(srv.Location.Blade),
			model.LabelName(chassisLabel):        model.LabelValue(srv.Location.Chassis),
			model.LabelName(clusterLabel):        model.LabelValue(srv.Location.Cluster),
			model.LabelName(zoneLabel):           model.LabelValue(srv.Location.ZoneID),
		},
	}
}

func (d *scwDiscoverer) getTargets() ([]*targetgroup.Group, error) {
	now := time.Now()
	srvs, err := d.client.GetServers(false, 0)
	requestDuration.Observe(time.Since(now).Seconds())
	if err != nil {
		requestFailures.Inc()
		return nil, err
	}

	level.Debug(d.logger).Log("msg", "get servers", "nb", len(*srvs))

	current := make(map[string]struct{})
	tgs := make([]*targetgroup.Group, len(*srvs))
	for _, s := range *srvs {
		tg := d.createTarget(&s)
		level.Debug(d.logger).Log("msg", "server added", "source", tg.Source)
		current[tg.Source] = struct{}{}
		tgs = append(tgs, tg)
	}

	// Add empty groups for servers which have been removed since the last refresh.
	for k := range d.lasts {
		if _, ok := current[k]; !ok {
			level.Debug(d.logger).Log("msg", "server deleted", "source", k)
			tgs = append(tgs, &targetgroup.Group{Source: k})
		}
	}
	d.lasts = current

	return tgs, nil
}

func (d *scwDiscoverer) Run(ctx context.Context, ch chan<- []*targetgroup.Group) {
	for c := time.Tick(time.Duration(d.refresh) * time.Second); ; {
		tgs, err := d.getTargets()
		if err == nil {
			ch <- tgs
		}

		// Wait for ticker or exit when ctx is closed.
		select {
		case <-c:
			continue
		case <-ctx.Done():
			return
		}
	}
}

func main() {
	a.HelpFlag.Short('h')

	a.Version(version.Print("prometheus-scaleway-sd"))

	_, err := a.Parse(os.Args[1:])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	logger := &scwLogger{
		log.With(
			log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout)),
			"ts", log.DefaultTimestampUTC,
			"caller", log.DefaultCaller,
		),
	}

	if *token == "" && *tokenf == "" {
		fmt.Println("need to pass --scw.token or --scw.token-file")
		os.Exit(1)
	}
	if *tokenf != "" {
		if *token != "" {
			fmt.Println("cannot pass --scw.token and --scw.token-file at the same time")
			os.Exit(1)
		}
		b, err := ioutil.ReadFile(*tokenf)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		*token = strings.TrimSpace(strings.TrimRight(string(b), "\n"))
	}

	client, err := api.NewScalewayAPI(
		*organization,
		*token,
		"Prometheus/SD-Agent",
		*region,
		func(s *api.ScalewayAPI) {
			s.Logger = logger
		},
	)
	if err != nil {
		fmt.Println("failed to create Scaleway API client:", err)
		os.Exit(1)
	}
	err = client.CheckCredentials()
	if err != nil {
		fmt.Println("failed to check Scaleway credentials:", err)
		os.Exit(1)
	}

	ctx := context.Background()
	disc := &scwDiscoverer{
		client:    client,
		port:      *port,
		refresh:   *refresh,
		separator: ",",
		logger:    logger,
		lasts:     make(map[string]struct{}),
	}
	sdAdapter := NewAdapter(ctx, *outputf, "scalewaySD", disc, logger)
	sdAdapter.Run()

	level.Debug(logger).Log("msg", "listening for connections", "addr", *listen)
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{ErrorLog: logger}))
	if err := http.ListenAndServe(*listen, nil); err != nil {
		level.Debug(logger).Log("msg", "failed to listen", "addr", *listen, "err", err)
		os.Exit(1)
	}
}
