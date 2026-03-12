.PHONY: build test lint clean install

BINARY := notes

build:
	go build -o $(BINARY) .

test:
	go test ./...

lint:
	golangci-lint run

clean:
	rm -f $(BINARY)

install:
	go install .
