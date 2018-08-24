A service discovery for the [Scaleway](https://www.scaleway.com/) cloud platform compatible with [Prometheus](https://prometheus.io).

Credits to [Simon Pasquier](https://github.com/simonpasquier) who wrotes most of the code on his own repository before merging it into this one.

## How it works

This service gets the list of servers from the Scaleway API and generates a file which is compatible with the Prometheus `file_sd` mechanism.

## Pre-requisites

You need your Scaleway secret key (token). You can create this token [in the console](https://cloud.scaleway.com/#/credentials).

## Installing it

Download the binary from the [Releases](https://github.com/simonpasquier/prometheus-scaleway-sd/releases) page.

## Running it

```
usage: sd adapter usage --scw.token=SCW.TOKEN [<flags>]

Tool to generate Prometheus file_sd target files for Scaleway.

Flags:
  -h, --help                    Show context-sensitive help (also try --help-long and --help-man).
      --output.file="scw.json"  The output filename for file_sd compatible file.
      --scw.organization=SCW.ORGANIZATION
                                The Scaleway organization.
      --scw.region="par1"       The Scaleway region. Leaving blank will fetch from all the regions.
      --scw.token=""            The authentication token (secret key).
      --scw.token-file=""       The authentication token file.
      --target.refresh=30       The refresh interval (in seconds).
      --target.port=80          The default port number for targets.
      --web.listen-address=":9465"
                                The listen address.
      --version                 Show application version.
```

## Integration with Prometheus

Here is a Prometheus `scrape_config` snippet that configures Prometheus to scrape node_exporter assuming that it is deployed on all your Scaleway servers.

```yaml
- job_name: node

  # Assuming that prometheus and prometheus-scaleway-sd are started from the same directory.
  file_sd_configs:
  - files: [ "./scw.json" ]

  # The relabeling does the following:
  # - overwrite the scrape address with the node_exporter's port.
  # - strip leading commas from the tags label.
  # - save the region label (par1/ams1).
  # - overwrite the instance label with the server's name.
  relabel_configs:
  - source_labels: [__meta_scaleway_private_ip]
    replacement: "${1}:9100"
    target_label: __address__
  - source_labels: [__meta_scaleway_tags]
    regex: ",(.+),"
    target_label: tags
  - source_labels: [__meta_scaleway_location_zone_id]
    target_label: region
  - source_labels: [__meta_scaleway_name]
    target_label: instance
```

The following meta labels are available on targets during relabeling:

* `__meta_scaleway_architecture`: the architecture of the server.
* `__meta_scaleway_blade_id`: the identifier of the blade (can be empty).
* `__meta_scaleway_chassis_id`: the identifier of the chassis (can be empty).
* `__meta_scaleway_cluster_id`: the identifier of the cluster (can be empty).
* `__meta_scaleway_commercial_type`: the commercial type of the server (eg START1-XS).
* `__meta_scaleway_hypervisor_id`: the identifier of the hypervisor.
* `__meta_scaleway_identifier`: the identifier of the server.
* `__meta_scaleway_image_id`: the identifier of the server's image.
* `__meta_scaleway_image_name`: the name of the server's image.
* `__meta_scaleway_name`: the name of the server.
* `__meta_scaleway_node_id`: the identifier of the node.
* `__meta_scaleway_organization`: the organization owning the server.
* `__meta_scaleway_platform_id`: the identifier of the platform.
* `__meta_scaleway_private_ip`: the private IP address of the server.
* `__meta_scaleway_public_ip`: the public IP address of the server (can be empty).
* `__meta_scaleway_state`: the state of the server.
* `__meta_scaleway_tags`: comma-separated list of tags associated to the server (trailing commas on both sides).
* `__meta_scaleway_zone_id`: the identifier of the zone (region).


## Contributing

PRs and issues are welcome.

## License

Apache License 2.0, see [LICENSE](https://github.com/simonpasquier/prometheus-scaleway-sd/blob/master/LICENSE).
