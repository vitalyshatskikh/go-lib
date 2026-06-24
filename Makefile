GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

API_CONFIG_PATH=$(shell pwd)/local.yml

.PHONY: setup-tools
#? setup-tools: Install dev tools
setup-tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
	go install golang.org/x/tools/cmd/goimports@latest
	go install gotest.tools/gotestsum@latest

.PHONY: test-unit
#? test-unit: Run the unit tests
test-unit:
	GOOS=$(GOOS) GOARCH=$(GOARCH) gotestsum --junitfile=coverage-unit.xml --jsonfile=coverage-unit.json -- \
 		-coverprofile=coverage-unit.txt -covermode atomic -race  ./...

.PHONY: fmt
#? fmt: Run gofmt
fmt:
	gofmt -s -l -w examples/ closer/ config/ http/ observability/

.PHONY: lint
#? lint: Run golangci-lint
lint:
	golangci-lint run ./...

.PHONY: run-example-restapi
#? run-example-restapi: Run golangci-lint
run-example-restapi:
	go run ./examples/restapi/*.go