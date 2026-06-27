APP_NAME := troll
BIN_DIR := .

.PHONY: build clean

build:
	go build -trimpath -ldflags="-s -w" -o $(BIN_DIR)/$(APP_NAME).exe .

clean:
	rm -f $(BIN_DIR)/$(APP_NAME).exe
