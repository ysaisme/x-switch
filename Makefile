.PHONY: build build-web build-all run clean tidy test install release

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS = -s -w -X main.version=$(VERSION)
GO ?= $(HOME)/.local/go/bin/go

build-web:
	cd web && npm run build
	@echo "copying web_dist to cmd/mswitch/web_dist..."
	rm -rf cmd/mswitch/web_dist
	cp -r web_dist cmd/mswitch/web_dist

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/mswitch ./cmd/mswitch

build-all: build-web build
	@echo "built bin/mswitch ($(VERSION), $(shell ls -lh bin/mswitch | awk '{print $$5}'))"

run: build
	./bin/mswitch start

clean:
	rm -rf bin/ web_dist/ cmd/mswitch/web_dist

tidy:
	$(GO) mod tidy

test:
	$(GO) test ./...

install: build
	cp bin/mswitch /usr/local/bin/mswitch

PLATFORMS = darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64

release: build-web
	@mkdir -p release
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*}; \
		GOARCH=$${platform#*/}; \
		ext=""; \
		if [ "$$GOOS" = "windows" ]; then ext=".exe"; fi; \
		echo "building $$GOOS/$$GOARCH..."; \
		GOOS=$$GOOS GOARCH=$$GOARCH CGO_ENABLED=1 $(GO) build -ldflags "$(LDFLAGS)" \
			-o release/mswitch-$(VERSION)-$$GOOS-$$GOARCH$$ext ./cmd/mswitch; \
	done
	@echo "release builds done in release/"

dev:
	cd web && npm run dev
