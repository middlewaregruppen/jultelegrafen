BINARY := jultelegrafen

.PHONY: all build test clean

all: build

build:
	go build -o bin/$(BINARY) ./...

test:
	go test ./...

clean:
	rm -rf bin/
