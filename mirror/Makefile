APP_NAME := mirror
BIN_DIR := .

BIN_SUFFIX := $(shell go env GOEXE)

.PHONY: build clean init

build:
	go build -trimpath -ldflags="-s -w" -o $(BIN_DIR)/$(APP_NAME)$(BIN_SUFFIX) ./cmd/mirror

init: build
	./$(APP_NAME)$(BIN_SUFFIX) init

clean:
	rm -f $(BIN_DIR)/$(APP_NAME)$(BIN_SUFFIX)
	rm -rf data/
