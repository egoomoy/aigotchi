.PHONY: build test run clean

build:
	go build -o bin/aigotchi ./cmd/aigotchi

test:
	go test ./... -v

run: build
	./bin/aigotchi

clean:
	rm -rf bin/
