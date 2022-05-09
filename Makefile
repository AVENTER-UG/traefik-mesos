#Dockerfile vars

#vars
IMAGENAME=traefik_mesos
TAG=v2.6.6
BRANCH=`git rev-parse --abbrev-ref HEAD`
IMAGEFULLNAME=avhost/${IMAGENAME}
BUILDDATE=`date -u +%Y-%m-%d`

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

build: 
	@echo ">>>> Build traefik executable"
	@if [ ! -d "traefik_repo" ] ; then \
		git clone git@github.com:traefik/traefik.git traefik_repo; \
	fi
	cd traefik_repo;	git checkout $(TAG)	
	patch -u traefik_repo/pkg/config/static/static_config.go -i static_config.patch
	patch -u traefik_repo/pkg/provider/aggregator/aggregator.go -i aggregator.patch
	cp -pr mesos traefik_repo/pkg/provider/
	@cd traefik_repo; go get -d
	@cd traefik_repo; go get github.com/mesos/mesos-go/api/v0/detector/zoo
	@cd traefik_repo; go mod vendor
	@cd traefik_repo; go mod tidy
	cd traefik_repo; $(MAKE) generate-webui
	cp static/mesos.svg traefik_repo/webui/static/statics/providers/
	cd traefik_repo; $(MAKE)

build-docker: build
	@echo ">>>> Build docker image"
	docker buildx build  --build-arg VERSION=${TAG} -t ${IMAGEFULLNAME}:${TAG} .
	docker buildx build  --build-arg VERSION=${TAG} -t ${IMAGEFULLNAME}:${TAG}-${BUILDDATE} .
	docker buildx build  --build-arg VERSION=${TAG} -t ${IMAGEFULLNAME}:latest .

publish:
	@echo ">>>> Publish it to repo"
	docker push ${IMAGEFULLNAME}:${TAG}
	docker push ${IMAGEFULLNAME}:latest 

clean:
	rm -rf traefik_repo



all: build build-docker clean
