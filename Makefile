.PHONY: build run test clean fmt lint

BINARY=httpctl
BUILD_DIR=bin

build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY) ./cmd/httpctl

run: build
	./$(BUILD_DIR)/$(BINARY)

test:
	go test -v -race -cover ./...

fmt:
	go fmt ./...

lint:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out

install: build
	cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)
