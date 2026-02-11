.PHONY: all web cli clean

all: web cli

# Build the single-file HTML converter
web:
	cd web && npm ci && npx webpack --config webpack.config.js
	@echo "Built: web/dist/index.html"

# Build the CLI for the current platform
cli:
	cd cli && go build -o heic2jpg .
	@echo "Built: cli/heic2jpg"

# Build CLI for Windows (requires mingw-w64 on Linux, or run on Windows)
cli-windows:
	cd cli && CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go build -o heic2jpg.exe .
	@echo "Built: cli/heic2jpg.exe"

clean:
	rm -rf web/dist cli/heic2jpg cli/heic2jpg.exe
