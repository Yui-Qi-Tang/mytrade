GITCOMMIT:=$(shell git describe --always)
BIN=$(notdir $(CURDIR))
BLD=bin

# GO
GO=go
GOFLAGS=-ldflags '-w -s -X main.GitCommit=$(GITCOMMIT)'
GOOPTIONS=-a -installsuffix cgo

# implementation
.PHONY: clean

all: pre_check test test_race build build_client

build:
	CGO_ENABLED=1 $(GO) build $(GOFLAGS) $(GOOPTIONS) -o $(BLD)/$(BIN)

build_client:
	CGO_ENABLED=1 $(GO) build $(GOFLAGS) $(GOOPTIONS) -o $(BLD)/$(BIN)-client service/client/client.go

test:
	$(GO) test ./...

test_race:
	$(GO) test -race ./...

pre_check:
	@if ! test -f $(BLD); then mkdir -p $(BLD); fi

clean:
	rm -f $(BLD)/* core