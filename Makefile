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
