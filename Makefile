ifeq ($(shell uname),Darwin)
	BINDIR = binaries/darwin
else ifeq ($(shell uname),Linux)
	BINDIR = binaries/linux_x86_64
endif

PATH := $(shell pwd)/$(BINDIR):$(PATH)

LDFLAGS = -ldflags "-X main.gitSHA=$(shell git rev-parse HEAD)"

.PHONY: all
all: build test

.PHONY: install-deps
install-deps:
	glide install

.PHONY: build
build:
	if [ ! -d bin ]; then mkdir bin; fi
	if [ ! -d logs ]; then mkdir logs; fi
	go build -o bin/manifest-generator $(LDFLAGS) main/main.go

.PHONY: fmt
fmt:
	find . -not -path "./vendor/*" -name '*.go' -type f | sed 's#\(.*\)/.*#\1#' | sort -u | xargs -n1 -I {} bash -c "cd {} && goimports -w *.go && gofmt -w -s -l *.go"

.PHONY: test
test:
ifndef BINDIR
	$(error Unable to set PATH based on current platform.)
endif
	#TODO go test $(V) ./handlers

.PHONY: clean
clean:
	go clean
	rm -f bin/manifest-generator
	rm -f logs/*

#.PHONY: circle-docker-build-push
#circle-docker-build-push:
#	docker login -e="." -u="$(QUAY_USER)" -p="$(QUAY_TOKEN)" quay.io
#	./docker_build.sh -p

#.PHONY: circle-fleet-deploy
#circle-fleet-deploy:
#	./fleet/circle-fleet-deploy.sh

#.PHONY: docker-build
#docker-build:
#	./docker_build.sh -r

#.PHONY: docker-build-push
#docker-build-push:
#	./docker_build.sh -r -p

#.PHONY: docker-image
#docker-image:
#	docker build -t alive-streamer .
#	$(foreach REGION,$(ECR_REGIONS),docker tag alive-streamer ???.dkr.ecr.$(REGION).amazonaws.com/playback/alive-streamer:$(CIRCLE_SHA1);)

#.PHONY: docker-push
#docker-push:
#	$(foreach REGION,$(ECR_REGIONS),docker push ???.dkr.ecr.$(REGION).amazonaws.com/playback/alive-streamer:$(CIRCLE_SHA1);)
