FROM golang:1.18 as build-stage

WORKDIR /src
COPY . .

WORKDIR /src/go
RUN make test
RUN make build
RUN ./resource_model_exporter --help

FROM quay.io/prometheus/busybox-linux-amd64:glibc AS bin
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

COPY --from=build-stage /src/go/resource_model_exporter /
COPY go/resources/ /resources/
RUN mkdir /output/
RUN chown "nobody:nobody" /output/
RUN chmod +x /resource_model_exporter

USER nobody
# free port see https://github.com/prometheus/prometheus/wiki/Default-port-allocations
EXPOSE 9901

ENTRYPOINT [ "/resource_model_exporter" ]
