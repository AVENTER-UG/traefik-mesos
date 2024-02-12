FROM alpine:latest

ENV ARCH amd64

ADD traefik_repo/dist/linux/${ARCH}/traefik /traefik
ADD entrypoint.sh /entrypoint.sh

RUN apk update
RUN apk add bash

WORKDIR /data
CMD "/traefik --configfile /data/traefik.toml"
