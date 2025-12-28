.PHONY: build test fmt format-check clean install help

BINARY_NAME=cclogs
BIN_DIR=bin
MAIN_PATH=./cmd/cclogs

## build: Build the binary
build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) $(MAIN_PATH)

## test: Run all tests
test:
	go test -v ./...

## fmt: Format all Go code
fmt:
	go fmt ./...

## format-check: Check if code is formatted (for CI)
format-check:
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "The following files are not formatted:"; \
		gofmt -l .; \
		exit 1; \
	fi

## clean: Remove built binaries
clean:
	rm -rf $(BIN_DIR)
	go clean

## install: Install the binary to $GOPATH/bin
install:
	go install $(MAIN_PATH)

## help: Display this help message
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'
