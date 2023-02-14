.PHONY: build test test-integration

build:
	go build -o conduit-connector-notion cmd/connector/main.go

download:
	@echo Download go.mod dependencies
	@go mod download

install-tools: download
	@echo Installing mockgen
	@go install github.com/golang/mock/mockgen
	@go mod tidy

generate:
	go generate ./...

test:
	go test $(GOTEST_FLAGS) -v -race ./...
