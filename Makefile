.PHONY: build-server build-cli run-server run-cli clean

build-server:
	go build -o bin/timer-server cmd/server/main.go

build-cli:
	go build -o bin/timer-cli cmd/cli/main.go

run-server: build-server
	./bin/timer-server

run-cli: build-cli
	./bin/timer-cli

clean:
	rm -f bin/timer-server bin/timer-cli
