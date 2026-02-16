.PHONY: build run test clean

build:
	go build -o bin/webserver ./cmd/webserver

run: build
	./bin/webserver -port 8087

test:
	go vet ./...

clean:
	rm -rf bin/
