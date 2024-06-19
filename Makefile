#Dockerfile vars

#vars
IMAGENAME=traefik_mesos
TAG=v3.0.3
BRANCH=$(shell git symbolic-ref --short HEAD | xargs basename)
BRANCHSHORT=$(shell echo ${BRANCH} | awk -F. '{ print $$1"."$$2 }')
IMAGEFULLNAME=avhost/${IMAGENAME}
BUILDDATE=$(shell date -u +%Y%m%d)
VERSION_TU=$(subst -, ,$(TAG:v%=%))	
BUILD_VERSION=$(word 1,$(VERSION_TU))
LASTCOMMIT=$(shell git log -1 --pretty=short | tail -n 1 | tr -d " " | tr -d "UPDATE:")

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
        BRANCHSHORT=latest
endif

clone: 
	@if [ ! -d "traefik_repo" ] ; then \
		git clone https://github.com/traefik/traefik.git traefik_repo; \
	fi
	cd traefik_repo;	git checkout $(TAG)

patch:	
	patch -u traefik_repo/pkg/config/static/static_config.go -i static_config.patch
	patch -u traefik_repo/pkg/provider/aggregator/aggregator.go -i aggregator.patch
	cp -pr mesos traefik_repo/pkg/provider/

build: 
	@echo ">>>> Build traefik executable ${BUILD_VERSION}"
	cp -pr mesos traefik_repo/pkg/provider/
	@cd traefik_repo; go get -d 
	@cd traefik_repo; go get github.com/mesos/mesos-go/api/v0/detector/zoo
	@cd traefik_repo; go mod tidy
	cd traefik_repo; $(MAKE) generate-webui
	cp static/mesos.svg traefik_repo/webui/static/providers/
	export VERSION=${BUILD_VERSION}; cd traefik_repo; $(MAKE)

build-docker: build
	@echo ">>>> Build docker image" ${BRANCH}
	docker build -t ${IMAGEFULLNAME}:latest . 

push: build
	@echo ">>>> Publish it to repo" ${BRANCH}_${BUILDDATE}
	docker buildx create --use --name buildkit
	docker buildx build --platform linux/amd64 --push --build-arg VERSION=${TAG} -t ${IMAGEFULLNAME}:${BRANCH} .
	docker buildx build --platform linux/amd64 --push --build-arg VERSION=${TAG} -t ${IMAGEFULLNAME}:${BRANCHSHORT} .
	docker buildx build --platform linux/amd64 --push --build-arg VERSION=${TAG} -t ${IMAGEFULLNAME}:latest .
	docker buildx rm buildkit

clean:
	rm -rf traefik_repo



all: clone patch build-docker
