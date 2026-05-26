BINARY  := pitlist
MODULE  := github.com/roramirez/pitlist
BUILD_DIR := ./bin
CMD     := ./cmd/pitlist
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: all build install run test vet fmt lint verify vuln check clean help

all: help

help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  build    Compile the binary"
	@echo "  install  Install the binary with go install"
	@echo "  run      Run the app without installing"
	@echo "  test     Run tests with race detection"
	@echo "  vet      Run go vet"
	@echo "  fmt      Format source files with gofmt"
	@echo "  lint     Run golangci-lint (implies vet)"
	@echo "  verify   Verify module dependencies"
	@echo "  vuln     Check for known vulnerabilities"
	@echo "  check    Run fmt, vet, verify, and test"
	@echo "  clean    Remove built binary"

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) $(CMD)

install:
	go install $(LDFLAGS) $(CMD)

run:
	go run $(LDFLAGS) $(CMD) $(filter-out $@,$(MAKECMDGOALS))

%:
	@:

test:
	go test -mod=readonly ./... -race

vet:
	go vet ./...

fmt:
	gofmt -w ./...

lint: vet
	golangci-lint run ./...

verify:
	go mod verify

vuln:
	govulncheck ./...

check: fmt lint verify test
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

check: fmt vet verify test

clean:
	rm -rf $(BUILD_DIR)
