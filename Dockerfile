FROM alpine:latest

ADD traefik_repo/dist/traefik /traefik
ADD entrypoint.sh /entrypoint.sh

RUN apk update
RUN apk add bash

ENTRYPOINT /entrypoint.sh
