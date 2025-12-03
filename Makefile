GOBIN ?= $(shell go env GOPATH)/bin

.PHONY: lint fmt test

fmt:
	gofmt -w -s .

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed. Install via: brew install golangci-lint"; exit 1; }
	golangci-lint run

test:
	go test ./...
