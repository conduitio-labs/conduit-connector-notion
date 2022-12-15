.PHONY: build test test-integration

build:
	go build -o conduit-connector-notion cmd/connector/main.go

download:
	@echo Download go.mod dependencies
	@go mod download

install-tools: download
	@echo Installing tools from tools.go
	@go list -f '{{ join .Imports "\n" }}' tools.go | xargs -tI % go install %
	@go mod tidy

generate:
	go generate ./...

test:
	go test $(GOTEST_FLAGS) -v -race ./...
