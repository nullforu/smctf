SHELL := /bin/bash

GO ?= go

.PHONY: all fmt vet lint test build

all: lint test build

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

lint: vet

test:
	$(GO) test ./...

build:
	$(GO) build ./cmd/server

run:
	$(GO) run ./cmd/server