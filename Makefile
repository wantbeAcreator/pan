.PHONY: all clean

all: dist/pan-linux-amd64 dist/pan-windows-amd64.exe

dist/pan-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/pan-linux-amd64 .

dist/pan-windows-amd64.exe:
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/pan-windows-amd64.exe .

clean:
	rm -rf dist
