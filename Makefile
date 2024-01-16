BUILDDIR ?= $(CURDIR)/build
NAMESPACE := ghcr.io/b2network
PROJECT := b2-indexer
DOCKER_IMAGE := $(NAMESPACE)/$(PROJECT)
COMMIT_HASH := $(shell git rev-parse --short=7 HEAD)
DATE=$(shell date +%Y%m%d-%H%M%S)
DOCKER_TAG := ${DATE}-$(COMMIT_HASH)

###############################################################################
###                                  Build                                  ###
###############################################################################

BUILD_TARGETS := build install

build: BUILD_ARGS=-o $(BUILDDIR)/
build-linux:
	GOOS=linux GOARCH=amd64 LEDGER_ENABLED=false $(MAKE) build

$(BUILD_TARGETS): go.sum $(BUILDDIR)/
	go $@ $(BUILD_FLAGS) $(BUILD_ARGS) ./...

$(BUILDDIR)/:
	mkdir -p $(BUILDDIR)/

image-build:
	docker build -t ${DOCKER_IMAGE}:${DOCKER_TAG} .

image-push:
	docker push --all-tags ${DOCKER_IMAGE}

image-list:
	docker images | grep ${DOCKER_IMAGE}

$(MOCKS_DIR):
	mkdir -p $(MOCKS_DIR)

distclean: clean tools-clean

clean:
	rm -rf \
    $(BUILDDIR)/ \
    artifacts/ \
    tmp-swagger-gen/

all: build

build-all: tools build lint test vulncheck

.PHONY: distclean clean build-all