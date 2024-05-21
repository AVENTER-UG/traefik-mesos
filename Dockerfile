FROM alpine:3.19
LABEL maintainer="Andreas Peters <support@aventer.biz>"
LABEL org.opencontainers.image.title="traefik-mesos" 
LABEL org.opencontainers.image.description="Traefik Proxy/Loadbalancer with Apache Mesos/ClusterD Provider"
LABEL org.opencontainers.image.vendor="AVENTER UG (haftungsbeschränkt)"
LABEL org.opencontainers.image.source="https://github.com/AVENTER-UG/"

ENV ARCH amd64

ADD traefik_repo/dist/linux/${ARCH}/traefik /traefik
ADD entrypoint.sh /entrypoint.sh

RUN apk update
RUN apk add bash

WORKDIR /data
CMD "/traefik --configfile /data/traefik.toml"
