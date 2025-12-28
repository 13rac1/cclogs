.PHONY: build test fmt format-check clean install release release-dry-run help

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

## release: Create a new release (example: make release VERSION=v1.0.0)
release:
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required. Usage: make release VERSION=v1.0.0"; exit 1; fi
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)

## release-dry-run: Test release process locally
release-dry-run:
	goreleaser release --snapshot --clean

## help: Display this help message
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'
