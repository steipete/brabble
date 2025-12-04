GOBIN ?= $(shell go env GOPATH)/bin
WHISPER_INC ?= /usr/local/include/whisper
WHISPER_LIB ?= /usr/local/lib/whisper
CGO_CFLAGS ?= -I$(WHISPER_INC)
CGO_LDFLAGS ?= -L$(WHISPER_LIB)

.PHONY: lint fmt test build

fmt:
	gofmt -w -s .

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed. Install via: brew install golangci-lint"; exit 1; }
	DYLD_LIBRARY_PATH=$(WHISPER_LIB) CGO_CFLAGS='$(CGO_CFLAGS)' CGO_LDFLAGS='$(CGO_LDFLAGS)' golangci-lint run

test:
	DYLD_LIBRARY_PATH=$(WHISPER_LIB) CGO_CFLAGS='$(CGO_CFLAGS)' CGO_LDFLAGS='$(CGO_LDFLAGS)' go test ./...

build:
	CGO_CFLAGS='$(CGO_CFLAGS)' CGO_LDFLAGS='$(CGO_LDFLAGS)' go build -o bin/brabble ./cmd/brabble
	@command -v install_name_tool >/dev/null 2>&1 && install_name_tool -add_rpath $(WHISPER_LIB) bin/brabble || true
