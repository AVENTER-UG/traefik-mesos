#Dockerfile vars

#vars
IMAGENAME=traefik_mesos
TAG=v2.10.1
BRANCH=`git rev-parse --abbrev-ref HEAD`
IMAGEFULLNAME=avhost/${IMAGENAME}
BUILDDATE=`date -u +%Y-%m-%d`
VERSION_TU=$(subst -, ,$(TAG:v%=%))	
BUILD_VERSION=$(word 1,$(VERSION_TU))
LASTCOMMIT=$(shell git log -1 --pretty=short | tail -n 1 | tr -d " ")

.PHONY: help build build-docker clean all

help:
	    @echo "Makefile arguments:"
	    @echo ""
	    @echo "Makefile commands:"
	    @echo "build"
			@echo "build-docker"
	    @echo "all"
			@echo ${TAG}

.DEFAULT_GOAL := all

ifeq (${BRANCH}, master) 
        BRANCH=latest
endif

ifneq ($(shell echo $(LASTCOMMIT) | grep -E '^v|([0-9]+\.){0,2}(\*|[0-9]+)'),)
        BRANCH=${LASTCOMMIT}
else
        BRANCH=latest
endif

build: 
	@echo ">>>> Build traefik executable ${BUILD_VERSION}"
	@if [ ! -d "traefik_repo" ] ; then \
		git clone https://github.com/traefik/traefik.git traefik_repo; \
	fi
	cd traefik_repo;	git checkout $(TAG)	
	patch -u traefik_repo/pkg/config/static/static_config.go -i static_config.patch
	patch -u traefik_repo/pkg/provider/aggregator/aggregator.go -i aggregator.patch
	cp -pr mesos traefik_repo/pkg/provider/
	@cd traefik_repo; go get -d
	@cd traefik_repo; go get github.com/mesos/mesos-go/api/v0/detector/zoo
	@cd traefik_repo; go mod vendor
	cd traefik_repo; $(MAKE) generate-webui
	cp static/mesos.svg traefik_repo/webui/static/statics/providers/
	export VERSION=${BUILD_VERSION}; cd traefik_repo; $(MAKE)

build-docker: build
	@echo ">>>> Build docker image"
	docker build -t ${IMAGEFULLNAME}:${TAG} .
	docker build -t ${IMAGEFULLNAME}:latest . 

publish:
	@echo ">>>> Publish it to repo"
	docker buildx create --use --name buildkit
	docker buildx build --platform linux/arm64,linux/amd64 --push --build-arg VERSION=${TAG} -t ${IMAGEFULLNAME}:${TAG} .
	docker buildx build --platform linux/arm64,linux/amd64 --push --build-arg VERSION=${TAG} -t ${IMAGEFULLNAME}:${TAG}-${BUILDDATE} .
	docker buildx build --platform linux/arm64,linux/amd64 --push --build-arg VERSION=${TAG} -t ${IMAGEFULLNAME}:latest .
	docker buildx rm buildkit

clean:
	rm -rf traefik_repo



all: build build-docker publish clean
