# Resource Model Exporter

<!-- [![Docker Repository on Quay](https://quay.io/repository/prometheus/prometheus/status)][quay]
[![Docker Pulls](https://img.shields.io/docker/pulls/prom/prometheus.svg?maxAge=604800)][hub]
[![Go Report Card](https://goreportcard.com/badge/github.com/prometheus/prometheus)](https://goreportcard.com/report/github.com/prometheus/prometheus)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/486/badge)](https://bestpractices.coreinfrastructure.org/projects/486)
[![Gitpod ready-to-code](https://img.shields.io/badge/Gitpod-ready--to--code-blue?logo=gitpod)](https://gitpod.io/#https://github.com/prometheus/prometheus)
[![Fuzzing Status](https://oss-fuzz-build-logs.storage.googleapis.com/badges/prometheus.svg)](https://bugs.chromium.org/p/oss-fuzz/issues/list?sort=-opened&can=1&q=proj:prometheus) -->

<!-- Visit [prometheus.io](https://prometheus.io) for the full documentation,
examples and guides. -->

Resource Model Exporter is a Prometheus exporter.
It collects resource metrics (CPU/Memory usage output) and dimensioning metrics (application usage input) for a container (or application) from an existing Prometheus instance for a given time window, find the relationship between the input and the output, exposes the model as metrics that can be ingested by a Prometheus instance and graphed with Grafana.
The Metrics and more information about the measurements are also exported to json files for off-line analysis.

## Principles

Dimensioning (aka sizing) is the process of determining the optimal size of a system in terms of its resource requirements and system performance.
One of the key aspect is to be able to predict the CPU/Memory/Storage/IOPS footprint on which traditionally some engineering margins are added to find out the HW to procure. The name of the game is to have an accurate prediction of the usage so that the engineering margins and/or fudge factors are small in order to have an optimized HW cost.

Most of the time it is rather complex for the developer to know in advance the footprint of his application. And sometimes we are even using application that we did not write. The approach taken here is purely based on experience/measurement. We have a system with multiple containers/applications, we have a dedicated monitoring stack in charge of measuring both inputs and output usage. We want to find out the existing resource model for each of those containers without knowing the code underneath.

The running sequence of the resource model exporter for each container/application is to :
- measure the inputs variables/metrics (predictors) such as qty of transactions, qty of object retained in memory/disk
- measure the outputs variables/metrics (observed) such as CPU/Memory/Storage/IOPS
- check that there is no bottleneck achieved on CPU/Memory/Storage/IOPS axis (by checking the container limits, ...)
- run a linear regression (MLR) between the predictors and observed variables for a given load
- expose the resource model as prometheus metrics and Yaml output

The cherry on the cake is to visualize the delta between the model and the real current usage on a graph

<!-- ![](https://cdn.jsdelivr.net/gh/prometheus/prometheus@c34257d069c630685da35bcef084632ffd5d6209/documentation/images/architecture.svg) -->

## Configuration

There are 3 config files :
- predictors.yml
In which you need to specify the dimensioning inputs that will be measured

```
- name: kubestate
  vars:
  - name: container
    value: eaa-platform-kubestate-exporter
  - name: namespace
    value: ".*"
  - name: pod
    value: ".*"
  - name: node
    value: ".*"
  resources:
  - name: cpu
    predictors:
    - name: scraped_metrics
      query: sum(scrape_samples_scraped{job="kube-state-metrics-exporter"})
  - name: mem
    predictors:
    - name: scraped_metrics
      query: sum(scrape_samples_scraped{job="kube-state-metrics-exporter"})
```

- observed.yml
That is supposed to be generic for any container orchestrator based on kubernetes
```
- name: cpu
  unit: m
  query: 1000*sum (rate (container_cpu_usage_seconds_total{pod=~"$pod",namespace=~"$namespace",container=~"$container"}[$interval]))
- name: mem
  unit: Mi
  query: sum (container_memory_working_set_bytes{pod=~"$pod",namespace=~"$namespace",container=~"$container"})/(1024*1024)
```

- control.yml
That is supposed to be generic for any container orchestrator based on kubernetes
```
- name: cpu_limit
  unit: m
  query: 1000*max (kube_pod_container_resource_limits_cpu_cores{pod=~"$pod",namespace=~"$namespace",container=~"$container"})
- name: mem_limit
  unit: Mi
  query: max (kube_pod_container_resource_limits_memory_bytes{pod=~"$pod",namespace=~"$namespace",container=~"$container"})/(1024*1024)
- name: image_version
  query: topk(1,kube_pod_container_info{pod=~"$pod",container=~"$container"})
```

- ENV files
```
PROMETHEUS_ENDPOINT=http://prometheus:9090
PROMETHEUS_AUTH_USER=admin
PROMETHEUS_AUTH_PWD=admin
REGRESSION_MAX_ROI=7d
SAMPLING_INTERVAL=5m
MAIL_ORIGINATOR=originator@mycompany.com
MAIL_TO_AGGR=recipient@mycompany.com
MAIL_SMTP_PORT=25
MAIL_SMTP_USER=admin
MAIL_SMTP_PWD=admin
```

It is currently support up to 4 predictors and 4 polynomial degrees (quartic).

## HOT Reload
It is essential when modeling to do some try and fail on the predictors or any of the yaml config to improve the R² value.
As you do not want to restart the application especially when you deployed the resource model exporter as a container, the best is to use the SIGHUP magic !

$ ps auxwww | grep model_exp
xy       28027  1.8  0.1 1308792 21188 pts/3   Sl+  11:43   0:00 ./resource_model_exporter

$ kill -HUP 28027

Then it will reload the yaml config

WARN[0144] HOT RELOAD 
INFO[0144] Successfully Opened resources/predictors.yml 
INFO[0144] Successfully Opened resources/observed.yml 
INFO[0144] Successfully Opened resources/info.yml 
INFO[0144] Successfully Opened resources/limits.yml 
INFO[0144] Successfully Opened output/regressions.yml 
INFO[0144] Yaml Config Reloaded 

## Email

By default an email will be sent with all yaml files once a day (at midnight).
These could be use for off line analysis or for aggregations of the results.
You can disable that feature with flag --email=false

## Install

There are various ways of installing Resource Model Exporter.

### Precompiled binaries

Precompiled binaries for released versions are available in the
<!-- [*download* section](https://prometheus.io/download/)
on [prometheus.io](https://prometheus.io). Using the latest production release binary
is the recommended way of installing Prometheus.
See the [Installing](https://prometheus.io/docs/introduction/install/)
chapter in the documentation for all the details. -->

### Docker images

<!-- Docker images are available on [Quay.io](https://quay.io/repository/prometheus/prometheus) or [Docker Hub](https://hub.docker.com/r/prom/prometheus/).

You can launch a Prometheus container for trying it out with

    $ docker run --name prometheus -d -p 127.0.0.1:9090:9090 prom/prometheus

Prometheus will now be reachable at http://localhost:9090/. -->

### Building from source

To build Resource Model Exporter from source code, first ensure that have a working
Go environment with [version 1.14 or greater installed](https://golang.org/doc/install).
<!-- You also need [Node.js](https://nodejs.org/) and [Yarn](https://yarnpkg.com/)
installed in order to build the frontend assets.

You can directly use the `go` tool to download and install the `prometheus`
and `promtool` binaries into your `GOPATH`:

    $ GO111MODULE=on go get github.com/prometheus/prometheus/cmd/...
    $ prometheus --config.file=your_config.yml

*However*, when using `go get` to build Prometheus, Prometheus will expect to be able to
read its web assets from local filesystem directories under `web/ui/static` and
`web/ui/templates`. In order for these assets to be found, you will have to run Prometheus
from the root of the cloned repository. Note also that these directories do not include the
new experimental React UI unless it has been built explicitly using `make assets` or `make build`.

An example of the above configuration file can be found [here.](https://github.com/prometheus/prometheus/blob/main/documentation/examples/prometheus.yml)

You can also clone the repository yourself and build using `make build`, which will compile in
the web assets so that Prometheus can be run from anywhere:

    $ mkdir -p $GOPATH/src/github.com/prometheus
    $ cd $GOPATH/src/github.com/prometheus
    $ git clone https://github.com/prometheus/prometheus.git
    $ cd prometheus
    $ make build
    $ ./prometheus --config.file=your_config.yml -->

The Makefile provides several targets:

  * *build*: build the `resource_model_exporter`
  * *test*: run the tests
  <!-- * *test-short*: run the short tests
  * *format*: format the source code
  * *vet*: check the source code for common errors
  * *assets*: build the new experimental React UI -->

### Building the Docker image

The `make docker` target is designed for use in our CI system.
You can build a docker image locally with the following commands:
<!--
    $ make promu
    $ promu crossbuild -p linux/amd64
    $ make common-docker-amd64

*NB* if you are on a Mac, you will need [gnu-tar](https://formulae.brew.sh/formula/gnu-tar). -->


## More information

Today only cpu/mem limits are looked at, storage limits will follow once the feature is added to kubernetes 
https://github.com/kubernetes/kubernetes/issues/92287
  <!-- * The source code is periodically indexed: [Prometheus Core](https://godoc.org/github.com/prometheus/prometheus).
  * You will find a CircleCI configuration in [`.circleci/config.yml`](.circleci/config.yml).
  * See the [Community page](https://prometheus.io/community) for how to reach the Prometheus developers and users on various communication channels. -->

## Contributions

Pull requests, comments and suggestions are welcome.

Please see [CONTRIBUTING.md](CONTRIBUTING.md) for more information.d](https://github.com/prometheus/prometheus/blob/main/CONTRIBUTING.md) -->

