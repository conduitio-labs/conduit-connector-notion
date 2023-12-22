.PHONY: build test test-integration

build:
	go build -o conduit-connector-notion cmd/connector/main.go

test:
	go test $(GOTEST_FLAGS) -v -race ./...

.PHONY: install-tools
install-tools:
	@echo Installing tools from tools.go
	@go list -e -f '{{ join .Imports "\n" }}' tools.go | xargs -tI % go install %
	@go mod tidy

PHONY: lint
lint:
	golangci-lint run -v