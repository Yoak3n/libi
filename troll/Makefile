APP_NAME := troll
BIN_DIR := .

# Auto-detect OS-specific binary extension (.exe on Windows, none on Linux/macOS)
BIN_SUFFIX := $(shell go env GOEXE)
.PHONY: build clean

build:
	go build -trimpath -ldflags="-s -w" -o $(BIN_DIR)/$(APP_NAME)$(BIN_SUFFIX) .

clean:
	rm -f $(BIN_DIR)/$(APP_NAME)$(BIN_SUFFIX)
