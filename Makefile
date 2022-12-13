.PHONY: build test test-integration

build:
	go build -o conduit-connector-notion cmd/connector/main.go

test:
	go test $(GOTEST_FLAGS) -v -race ./...
