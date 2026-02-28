BINARY := starfix
BUILD_DIR := bin
CMD := ./cmd/starfix

.PHONY: build test lint vet clean install

build:
	go build -o $(BUILD_DIR)/$(BINARY) $(CMD)

test:
	go test ./...

lint:
	go vet ./...

vet:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR)

install: build
	cp $(BUILD_DIR)/$(BINARY) ~/.local/bin/$(BINARY)
