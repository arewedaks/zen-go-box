VERSION := $(shell git describe --tags --always 2>/dev/null || echo "v1.10.2-go")
LDFLAGS := -s -w -X main.version=$(VERSION)
GOBUILD := CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)"

.PHONY: all build-arm64 build-armv7 compress clean

all: build-arm64 build-armv7 compress

build-arm64:
	GOOS=linux GOARCH=arm64 $(GOBUILD) -o bin/zengobox-arm64 ./cmd/zengobox/

build-armv7:
	GOOS=linux GOARCH=arm GOARM=7 $(GOBUILD) -o bin/zengobox-armv7 ./cmd/zengobox/

compress:
	@if command -v upx >/dev/null 2>&1; then \
		echo "Compressing binaries with UPX..."; \
		upx --best --lzma bin/zengobox-arm64; \
		upx --best --lzma bin/zengobox-armv7; \
	else \
		echo "UPX not found, skipping compression."; \
	fi

clean:
	rm -rf bin/zengobox-*
	rm -rf bin/ZenGoBox-Magisk-*.zip

build-magisk: build-arm64 build-armv7
	@echo "Packaging Universal Magisk Module (ARM64 & ARMv7)..."
	@rm -f bin/ZenGoBox-Magisk-v1.0.12.zip
	@mkdir -p bin/magisk-temp
	@cp bin/zengobox-arm64 bin/magisk-temp/
	@cp bin/zengobox-armv7 bin/magisk-temp/
	@cp shell/module.prop bin/magisk-temp/
	@cp shell/customize.sh bin/magisk-temp/
	@cp shell/action.sh bin/magisk-temp/
	@cp shell/uninstall.sh bin/magisk-temp/
	@cp shell/service.sh bin/magisk-temp/
	@cp -r shell/webroot bin/magisk-temp/
	@cd bin/magisk-temp && zip -r9 ../ZenGoBox-Magisk-v1.0.12.zip .
	@rm -rf bin/magisk-temp
	@echo "Magisk Module ZIP created at bin/ZenGoBox-Magisk-v1.0.12.zip"
