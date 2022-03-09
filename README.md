# Apache Mesos provider for Traefik

These provider will add the functionality to use traefik with Apache Mesos.

## How to use the docker image?

```
docker run -p 80:80 -p 443:433 -p 9000:9000 -v <config_toml_directory>:/data:rw avhost/traefik_mesos:latest
```

## How to add the Mesos Provider

Edit your traefik.toml and include these configuration:

``` 
[providers.mesos]
endpoint = "<your_mesos_master>"
principal = "<mesos_username>"
secret = "<mesos_password>"
SSL = true
``` 

## How to add Traefik routes and services?

To tell traefik how it should handle the mesos tasks, we have to use traefik labels. 
As example:

``` 
    "traefik.enable": "true",
    "traefik.http.routers.homepage-ssl.tls": "true",
    "traefik.http.routers.homepage.entrypoints": "web",

    # The service object with the name "homepage-web" and "homepage-web-ssl" 
    # will be generated from the name of the Mesos Task (or Marathon) PortMapping object.
    "traefik.http.routers.homepage-ssl.service": "homepage-web-ssl",
    "traefik.http.routers.homepage.service": "homepage-web",

    "traefik.http.middlewares.homepage.redirectscheme.scheme": "https",
    "traefik.http.routers.homepage.rule": "Host(`your.example.com`)",
    "traefik.http.routers.homepage-ssl.rule": "Host(`your.example.com`)",
    "traefik.http.routers.homepage-ssl.tls.certresolver": "sslcert",
    "traefik.http.routers.homepage-ssl.entrypoints": "websecure"
```


<p align="center">
<img src="docs/content/assets/img/traefik.logo.png" alt="Traefik" title="Traefik" />
</p>

## Doku and links to the official traefik

[![Build Status SemaphoreCI](https://semaphoreci.com/api/v1/containous/traefik/branches/master/shields_badge.svg)](https://semaphoreci.com/containous/traefik)
[![Docs](https://img.shields.io/badge/docs-current-brightgreen.svg)](https://doc.traefik.io/traefik)
[![Go Report Card](https://goreportcard.com/badge/traefik/traefik)](https://goreportcard.com/report/traefik/traefik)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/traefik/traefik/blob/master/LICENSE.md)
[![Join the community support forum at https://community.traefik.io/](https://img.shields.io/badge/style-register-green.svg?style=social&label=Discourse)](https://community.traefik.io/)
[![Twitter](https://img.shields.io/twitter/follow/traefik.svg?style=social)](https://twitter.com/intent/follow?screen_name=traefik)


Traefik (pronounced _traffic_) is a modern HTTP reverse proxy and load balancer that makes deploying microservices easy.
Traefik integrates with your existing infrastructure components ([Docker](https://www.docker.com/), [Swarm mode](https://docs.docker.com/engine/swarm/), [Kubernetes](https://kubernetes.io), [Marathon](https://mesosphere.github.io/marathon/), [Consul](https://www.consul.io/), [Etcd](https://coreos.com/etcd/), [Rancher](https://rancher.com), [Amazon ECS](https://aws.amazon.com/ecs), ...) and configures itself automatically and dynamically.
Pointing Traefik at your orchestrator should be the _only_ configuration step you need.
