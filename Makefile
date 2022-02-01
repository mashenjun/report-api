SHELL := /bin/bash
PROJECT=report-api
GOPATH ?= $(shell go env GOPATH)
BIN_NAME := reportd

# Ensure GOPATH is set before running build process.
ifeq "$(GOPATH)" ""
  $(error Please set the environment variable GOPATH before running `make`)
endif
BUILD_FLAG			:= -trimpath
GOENV   	    	:= GO111MODULE=on CGO_ENABLED=0
GO                  := $(GOENV) go
GOBUILD             := $(GO) build $(BUILD_FLAG)
GOTEST              := $(GO) test -v --count=1 --parallel=1 -p=1
GORUN               := $(GO) run
TEST_LDFLAGS        := ""

PACKAGE_LIST        := go list ./...| grep -vE "cmd"
PACKAGES            := $$($(PACKAGE_LIST))

CURDIR := $(shell pwd)
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
GOLANGCI_LINT=$(shell pwd)/bin/revive
# Targets
.PHONY: server server-linux test run

# build server with local os and arch
server: lint
	$(GOBUILD) -o bin/$(BIN_NAME) .

# build server with os=linux and arch=amd64
server-linux: lint
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o bin/linux/report-api .

# run starts the server with dev config
run: server
	./bin/$(BIN_NAME) -c config.dev.yaml

test:
	$(GOTEST) ./...

# Run golangci-lint linter
lint: golangci-lint
	$(GOLANGCI_LINT) -config revive_lint.toml -formatter friendly ./...

golangci-lint: # Download golangci-lint locally if necessary
	$(call go-get-tool,$(GOLANGCI_LINT),github.com/mgechev/revive@master)

# go-get-tool will 'go get' any package $2 and install it to $1.
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

