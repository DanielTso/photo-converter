VERSION ?= dev
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  = -X github.com/DanielTso/pixshift/internal/version.Version=$(VERSION) \
           -X github.com/DanielTso/pixshift/internal/version.Commit=$(COMMIT) \
           -X github.com/DanielTso/pixshift/internal/version.Date=$(DATE)

.PHONY: build build-static clean test lint

# Build for current platform
build:
	CGO_ENABLED=1 go build -ldflags '$(LDFLAGS)' -o pixshift ./cmd/pixshift
	@echo "Built: pixshift"

# Build with static linking (Linux/Windows)
build-static:
	CGO_ENABLED=1 go build -ldflags '$(LDFLAGS) -extldflags "-static"' -o pixshift ./cmd/pixshift
	@echo "Built: pixshift (static)"

# Cross-compile for Windows (requires mingw-w64)
build-windows:
	CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 \
		go build -ldflags '$(LDFLAGS) -extldflags "-static"' -o pixshift.exe ./cmd/pixshift
	@echo "Built: pixshift.exe"

clean:
	rm -f pixshift pixshift.exe

test:
	go test ./...

lint:
	golangci-lint run ./...

# Legacy targets for backward compatibility with photo-converter
web:
	cd web && npm ci && npx webpack --config webpack.config.js
	@echo "Built: web/dist/index.html"
