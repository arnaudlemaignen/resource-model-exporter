FROM quay.io/prometheus/busybox-linux-amd64:glibc AS bin
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

COPY bin/resource_model_exporter /
COPY resources/ /resources/
RUN chmod +x /resource_model_exporter

# ENV PROM_LOGIN=admin:admin
# ENV PROM_ENDPOINT=http://prometheus:9090

USER nobody
# free port see https://github.com/prometheus/prometheus/wiki/Default-port-allocations
EXPOSE 9801

ENTRYPOINT [ "/resource_model_exporter" ]
