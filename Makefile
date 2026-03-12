.PHONY: build test lint clean install

BINARY := notes

build:
	go build -o $(BINARY) ./cmd/notes

test:
	go test ./...

lint:
	golangci-lint run

clean:
	rm -f $(BINARY)

install:
	go install ./cmd/notes
