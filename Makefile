.PHONY: build test lint clean install

BINARY := notes
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X github.com/dreikanter/notescli/internal/cli.Version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/notes

test:
	go test ./...

lint:
	go tool golangci-lint run

clean:
	rm -f $(BINARY)

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/notes

