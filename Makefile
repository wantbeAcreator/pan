.PHONY: all clean linux windows darwin darwin-native

all: linux

linux: dist/pan-linux-amd64

dist/pan-linux-amd64:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -ldflags="-s -w" -o dist/pan-linux-amd64 .

windows:
	@if command -v fyne-cross >/dev/null 2>&1; then \
		fyne-cross windows -arch=amd64 -app-id pan -ldflags="-s -w -H windowsgui"; \
		mkdir -p dist; \
		cp fyne-cross/dist/windows-amd64/pan.exe dist/pan-windows-amd64.exe; \
		echo "Done: dist/pan-windows-amd64.exe"; \
	elif command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1; then \
		GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -ldflags="-s -w -H windowsgui" -o dist/pan-windows-amd64.exe .; \
	else \
		echo "Install fyne-cross (go install github.com/fyne-io/fyne-cross@latest) or mingw-w64"; \
		exit 1; \
	fi

darwin-native:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -ldflags="-s -w" -o dist/pan-darwin-amd64 .

darwin:
	@if command -v fyne-cross >/dev/null 2>&1; then \
		fyne-cross darwin -arch=amd64 -app-id pan -ldflags="-s -w"; \
		mkdir -p dist; \
		cp fyne-cross/dist/darwin-amd64/pan dist/pan-darwin-amd64; \
		echo "Done: dist/pan-darwin-amd64"; \
	else \
		echo "Install fyne-cross: go install github.com/fyne-io/fyne-cross@latest"; \
		echo "Darwin cross-compile requires macOS SDK bundled via fyne-cross Docker."; \
		exit 1; \
	fi

clean:
	rm -rf dist fyne-cross
