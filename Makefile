.PHONY: build build-web build-all run clean tidy test install release app dmg dev

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS = -s -w -X main.version=$(VERSION)
GO ?= $(HOME)/.local/go/bin/go
APP_NAME = mswitch
APP_VERSION = $(shell echo $(VERSION) | sed 's/^v//')

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
	rm -rf bin/ web_dist/ cmd/mswitch/web_dist build/ release/

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

app: build-all
	@echo "building $(APP_NAME).app..."
	@rm -rf build/$(APP_NAME).app
	@mkdir -p build/$(APP_NAME).app/Contents/MacOS
	@mkdir -p build/$(APP_NAME).app/Contents/Resources
	@clang -o build/$(APP_NAME).app/Contents/MacOS/$(APP_NAME) scripts/App.m -framework Cocoa -framework WebKit -fobjc-arc
	@cp bin/mswitch build/$(APP_NAME).app/Contents/Resources/mswitch
	@cp scripts/icon.icns build/$(APP_NAME).app/Contents/Resources/icon.icns
	@cp scripts/icon-menu.png build/$(APP_NAME).app/Contents/Resources/icon-menu.png
	@sed -e 's/{{VERSION}}/$(APP_VERSION)/g' scripts/Info.plist > build/$(APP_NAME).app/Contents/Info.plist
	@chmod +x build/$(APP_NAME).app/Contents/MacOS/$(APP_NAME)
	@chmod +x build/$(APP_NAME).app/Contents/Resources/mswitch
	@codesign --force --deep --sign - build/$(APP_NAME).app
	@xattr -cr build/$(APP_NAME).app
	@echo "built build/$(APP_NAME).app"

dmg: app
	@echo "building $(APP_NAME).dmg..."
	@rm -f build/$(APP_NAME).dmg
	@mkdir -p build/dmg_temp
	@cp -R build/$(APP_NAME).app build/dmg_temp/
	@ln -sf /Applications build/dmg_temp/Applications
	@hdiutil create -volname "$(APP_NAME)" -srcfolder build/dmg_temp -ov -format UDZO build/$(APP_NAME).dmg
	@xattr -cr build/$(APP_NAME).dmg
	@rm -rf build/dmg_temp
	@echo "built build/$(APP_NAME).dmg ($(shell ls -lh build/$(APP_NAME).dmg | awk '{print $$5}'))"

dev:
	cd web && npm run dev
