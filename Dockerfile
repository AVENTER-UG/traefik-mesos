FROM alpine:latest

ADD traefik_repo/dist/traefik /traefik
ADD entrypoint.sh /entrypoint.sh

RUN apk update
RUN apk add bash

WORKDIR /data
CMD "/traefik --configfile traefik.toml"
