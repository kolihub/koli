# Common flags passed into Go's linker.
KUBECLIVERSION ?= unknown # glide.yaml
GITVERSION ?= unknown # git tag -l
GITCOMMIT ?= $(shell git rev-parse --short HEAD)
DATE ?= $(shell date -u "+%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := "-s -w \
-X github.com/kolihub/koli/pkg/cli/version.kubernetesClientVersion=${KUBECLIVERSION} \
-X github.com/kolihub/koli/pkg/cli/version.gitVersion=${GITVERSION} \
-X github.com/kolihub/koli/pkg/cli/version.gitCommit=${GITCOMMIT} \
-X github.com/kolihub/koli/pkg/cli/version.buildDate=${DATE}"

info:
	@echo "KUBECLIVERSION:   ${KUBECLIVERSION}"
	@echo "GITVERSION:       ${GITVERSION}"
	@echo "GITCOMMIT:        ${GITCOMMIT}"
	@echo "DATE:             ${DATE}"

build:
	mkdir -p ./build
	go build -ldflags ${LDFLAGS} -o build/koli-${GITVERSION} github.com/kolihub/koli/cmd
	go build -ldflags ${LDFLAGS} -o build/koli-controller-${GITVERSION} github.com/kolihub/cmd/controller

.PHONY: build
