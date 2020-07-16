
# Add bin dir to path
ifeq ($(shell uname),Darwin)
	BINDIR = binaries/darwin
else ifeq ($(shell uname),Linux)
	BINDIR = binaries/linux_x86_64
endif
PATH := $(shell pwd)/$(BINDIR):$(PATH)

BINARY_NAME="go-ts-segmenter"

# DOCKER Section
include ./secrets/docker-creds.secrets
DOCKER_IMAGE_NAME = docker-go-ts-segmenter
DOCKER_IMAGE_VERSION = 1.0

# Set flags for logs build
LDFLAGS = -ldflags "-X main.gitSHA=$(shell git rev-parse HEAD)"

.PHONY: build build_in_docker install_deps clean build_docker tag_latest_docker push_docker push_latest_docker last_built_date_docker shell_docker

build:
	if [ ! -d bin ]; then mkdir bin; fi
	if [ ! -d logs ]; then mkdir logs; fi
	go build -o "bin/${BINARY_NAME}" $(LDFLAGS) main.go

build_in_docker:
	go get
	if [ ! -d bin ]; then mkdir bin; fi
	if [ ! -d logs ]; then mkdir logs; fi
	go build -o "bin/${BINARY_NAME}" main.go

install_deps:
	go get

clean:
	go clean
	rm -f "bin/${BINARY_NAME}"
	rm -f logs/*

build_docker: Dockerfile
	docker build -t $(DOCKER_REPO_USER)/$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_VERSION) --rm .

tag_latest_docker:
	docker tag $(DOCKER_REPO_USER)/$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_VERSION) $(DOCKER_REPO_USER)/$(DOCKER_IMAGE_NAME):latest

push_docker:
	docker login -u $(DOCKER_REPO_USER) -p $(DOCKER_REPO_PASS)
	docker push $(DOCKER_REPO_USER)/$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_VERSION)
	docker logout

push_latest_docker:
	docker login -u $(DOCKER_REPO_USER) -p $(DOCKER_REPO_PASS)
	docker push $(DOCKER_REPO_USER)/$(DOCKER_IMAGE_NAME):latest
	docker logout

last_built_date_docker:
	docker inspect -f '{{ .Created }}' $(DOCKER_REPO_USER)/$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_VERSION)

shell_docker:
	docker run --rm -it --entrypoint /bin/bash $(DOCKER_REPO_USER)/$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_VERSION)

default: build