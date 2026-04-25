.PHONY: build test lint clean install update

BINARY := notesctl
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X github.com/dreikanter/notesctl/internal/cli.Version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/notesctl

test:
	go test ./...

lint:
	go tool golangci-lint run

clean:
	rm -f $(BINARY)

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/notesctl

update:
	git checkout main
	git pull --tags
	$(MAKE) install
	@echo "Installed: $$(notesctl --version)"

