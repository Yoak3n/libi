OUTPUT_DIR := bin

.PHONY: all clean bdanmu troll

all: bdanmu troll

bdanmu:
	cd bdanmu && wails3 build
	[ -d $(OUTPUT_DIR) ] || mkdir -p $(OUTPUT_DIR)
	cp bdanmu/bin/*.exe $(OUTPUT_DIR)/bdanmu.exe

troll:
	go build -trimpath -ldflags="-s -w" -o $(OUTPUT_DIR)/troll.exe ./troll

clean:
	rm -rf $(OUTPUT_DIR)
