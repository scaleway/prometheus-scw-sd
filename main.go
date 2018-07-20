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
	"net"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/prometheus/prometheus/documentation/examples/custom-sd/adapter"
	scwApi "github.com/scaleway/go-scaleway"
	scwTypes "github.com/scaleway/go-scaleway/types"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	a          = kingpin.New("sd adapter usage", "Tool to generate file_sd target files for unimplemented SD mechanisms.")
	outputFile = a.Flag("output.file", "Output file for file_sd compatible file.").Default("scw_sd.json").String()
	token      = a.Flag("token", "The token for Scaleway API.").Default("token").String()
	logger     log.Logger

	// addressLabel is the name for the label containing a target's address.
	addressLabel = model.MetaLabelPrefix + "scw_address"
	// srvLabel is the name for the label containing a target's server name.
	srvLabel = model.MetaLabelPrefix + "scw_node"
	// tagsLabel is the name of the label containing the tags assigned to the target.
	tagsLabel = model.MetaLabelPrefix + "scw_tags"
	// serviceIDLabel is the name of the label containing the service ID.
	serviceIDLabel = model.MetaLabelPrefix + "scw_id"
	// srvType is the name of the label containing the commercial type.
	srvType = model.MetaLabelPrefix + "scw_type"
	// srvArch is the name of the label containing the commercial arch.
	srvArch = model.MetaLabelPrefix + "scw_arch"
)

// Note: create a config struct for Scaleway SD type here.
type sdConfig struct {
	Token           string
	TagSeparator    string
	RefreshInterval int
}

// Discovery retrieves targets information from Scaleway API and updates them via watches.
type discovery struct {
	token           string
	refreshInterval int
	tagSeparator    string
	logger          log.Logger
}

func (d *discovery) appendScalewayServer(tgs []*targetgroup.Group, server scwTypes.ScalewayServer) []*targetgroup.Group {
	addr := net.JoinHostPort(server.PublicAddress.IP, fmt.Sprintf("%d", 9100))
	target := model.LabelSet{model.AddressLabel: model.LabelValue(addr)}
	// https://github.com/prometheus/prometheus/blob/master/documentation/examples/custom-sd/adapter-usage/main.go#L117
	tags := "," + strings.Join(server.Tags, ",") + ","
	labels := model.LabelSet{
		model.LabelName(srvArch):   model.LabelValue(server.Arch),
		model.LabelName(tagsLabel): model.LabelValue(tags),
		model.LabelName(srvType):   model.LabelValue(server.CommercialType),
		// model.AddressLabel:            model.LabelValue(addr),
		// model.LabelName(addressLabel): model.LabelValue(server.PublicAddress.IP),
		// model.LabelName(serviceIDLabel): model.LabelValue(server.Identifier),
	}
	for i := range tgs {
		if reflect.DeepEqual(tgs[i].Labels, labels) {
			tgs[i].Targets = append(tgs[i].Targets, target)
			return tgs
		}
	}
	tgroup := targetgroup.Group{
		Source: server.Name,
		Labels: make(model.LabelSet),
	}
	tgroup.Targets = make([]model.LabelSet, 0, 1)
	tgroup.Labels = labels
	tgroup.Targets = append(tgroup.Targets, target)
	tgs = append(tgs, &tgroup)
	return tgs
}

// Note: you must implement this function for your discovery implementation as part of the
// Discoverer interface. Here you should query your SD for it's list of known targets, determine
// which of those targets you care about, and then send those targets as a target.TargetGroup to the ch channel.
func (d *discovery) Run(ctx context.Context, ch chan<- []*targetgroup.Group) {
	for c := time.Tick(time.Duration(d.refreshInterval) * time.Second); ; {
		client, err := scwApi.NewScalewayAPI("", d.token, "", "")
		if err != nil {
			level.Error(d.logger).Log("msg", "Unable to create Scaleway API client", "err", err)
			time.Sleep(time.Duration(d.refreshInterval) * time.Second)
			continue
		}
		time.Sleep(time.Duration(2) * time.Second) // rate limit
		srvs, err := client.GetServers(true, 0)
		if err != nil {
			level.Error(d.logger).Log("msg", "Error retreiving server list", "err", err)
			time.Sleep(time.Duration(d.refreshInterval) * time.Second)
			continue
		}

		var tgs []*targetgroup.Group
		for _, srv := range *srvs {
			level.Info(d.logger).Log("msg", fmt.Sprintf("Found server: %s", srv.Name))
			tgs = d.appendScalewayServer(tgs, srv)
		}

		if err == nil {
			// We're returning all Scaleway services as a single targetgroup.
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

func newDiscovery(conf sdConfig) (*discovery, error) {
	cd := &discovery{
		token:           conf.Token,
		refreshInterval: conf.RefreshInterval,
		tagSeparator:    conf.TagSeparator,
		logger:          logger,
	}
	return cd, nil
}

func main() {
	a.HelpFlag.Short('h')

	_, err := a.Parse(os.Args[1:])
	if err != nil {
		fmt.Println("err: ", err)
		return
	}
	logger = log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)

	ctx := context.Background()

	// NOTE: create an instance of your new SD implementation here.
	cfg := sdConfig{
		TagSeparator:    ",",
		Token:           *token,
		RefreshInterval: 30,
	}

	disc, err := newDiscovery(cfg)
	if err != nil {
		fmt.Println("err: ", err)
	}
	sdAdapter := adapter.NewAdapter(ctx, *outputFile, "ScalewaySD", disc, logger)
	sdAdapter.Run()

	<-ctx.Done()
}
