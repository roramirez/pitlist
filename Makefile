BINARY = pitlist
BUILD_DIR = ./bin
CMD = ./cmd/pitlist

.PHONY: build install test lint fmt verify vuln check clean

build:
	go build -o $(BUILD_DIR)/$(BINARY) $(CMD)

install:
	go install $(CMD)

test:
	go test -mod=readonly ./... -race

lint:
	go vet ./...

fmt:
	gofmt -w ./...

verify:
	go mod verify

vuln:
	govulncheck ./...

check: fmt lint verify test

clean:
	rm -rf $(BUILD_DIR)
