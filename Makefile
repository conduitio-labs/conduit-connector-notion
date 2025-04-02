.PHONY: build
build:
	go build -o conduit-connector-notion cmd/connector/main.go

.PHONY: test
test:
	go test $(GOTEST_FLAGS) -v -race ./...

.PHONY: install-tools
install-tools:
	@echo Installing tools from tools/go.mod
	@go list -modfile=tools/go.mod tool | xargs -I % go list -modfile=tools/go.mod -f "%@{{.Module.Version}}" % | xargs -tI % go install %
	@go mod tidy

.PHONY: lint
lint:
	golangci-lint run
