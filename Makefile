BINARY = pitlist
BUILD_DIR = ./bin
CMD = ./cmd/pitlist

.PHONY: build install test lint clean

build:
	go build -o $(BUILD_DIR)/$(BINARY) $(CMD)

install:
	go install $(CMD)

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR)
