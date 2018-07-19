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
	// nodeLabel is the name for the label containing a target's node name.
	nodeLabel = model.MetaLabelPrefix + "scw_node"
	// tagsLabel is the name of the label containing the tags assigned to the target.
	tagsLabel = model.MetaLabelPrefix + "scw_tags"
	// serviceIDLabel is the name of the label containing the service ID.
	serviceIDLabel = model.MetaLabelPrefix + "scw_id"
	// nodeType is the name of the label containing the commercial type.
	nodeType = model.MetaLabelPrefix + "scw_type"
	// nodeArch is the name of the label containing the commercial arch.
	nodeArch = model.MetaLabelPrefix + "scw_arch"
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

func (d *discovery) parseServiceNodes(nodes []scwTypes.ScalewayServer, name string) *targetgroup.Group {
	tgroup := targetgroup.Group{
		Source: name,
		Labels: make(model.LabelSet),
	}
	tgroup.Targets = make([]model.LabelSet, 0, len(nodes))

	for _, node := range nodes {
		// We surround the separated list with the separator as well. This way regular expressions
		// in relabeling rules don't have to consider tag positions.
		var tags = "," + strings.Join(node.Tags, ",") + ","

		// If the service address is not empty it should be used instead of the node address
		// since the service may be registered remotely through a different node.
		var addr string
		addr = net.JoinHostPort(node.PublicAddress.IP, fmt.Sprintf("%d", 9100))
		target := model.LabelSet{model.AddressLabel: model.LabelValue(addr)}
		labels := model.LabelSet{
			model.AddressLabel:              model.LabelValue(addr),
			model.LabelName(addressLabel):   model.LabelValue(node.PublicAddress.IP),
			model.LabelName(nodeArch):       model.LabelValue(node.Arch),
			model.LabelName(tagsLabel):      model.LabelValue(tags),
			model.LabelName(serviceIDLabel): model.LabelValue(node.Identifier),
			model.LabelName(nodeType):       model.LabelValue(node.CommercialType),
		}
		tgroup.Labels = labels
		tgroup.Targets = append(tgroup.Targets, target)
	}
	return &tgroup
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
		var tgs []*targetgroup.Group
		srvs, err := client.GetServers(true, 0)
		if err != nil {
			level.Error(d.logger).Log("msg", "Error retreiving server list", "err", err)
			time.Sleep(time.Duration(d.refreshInterval) * time.Second)
			continue
		}
		tg := d.parseServiceNodes(*srvs, "Scaleway")
		tgs = append(tgs, tg)

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
