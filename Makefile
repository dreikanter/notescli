.PHONY: build test lint clean install tag

BINARY := notes
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X github.com/dreikanter/notescli/internal/cli.Version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/notes

test:
	go test ./...

lint:
	golangci-lint run

clean:
	rm -f $(BINARY)

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/notes

tag:
	@if [ -z "$(V)" ]; then echo "Usage: make tag V=0.5.0"; exit 1; fi
	git tag v$(V)
	git push origin v$(V)
