.PHONY: build build-all clean test install deps run snapshot npm-snapshot

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o nacos-mcp .

clean:
	rm -f nacos-mcp nacos-mcp-*
	rm -rf dist/
	go clean

test:
	go test -v ./...

deps:
	go mod download
	go mod tidy

run:
	go run .

build-all:
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o nacos-mcp-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o nacos-mcp-linux-arm64 .
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o nacos-mcp-darwin-amd64 .
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o nacos-mcp-darwin-arm64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o nacos-mcp-windows-amd64.exe .

# Local goreleaser dry-run
snapshot:
	goreleaser release --snapshot --clean --skip=publish

install: build
	cp nacos-mcp /usr/local/bin/

# Local npm dry-run: build snapshot binaries + assemble 7 npm packages (no publish)
npm-snapshot: snapshot
	@SNAP_VERSION=$$(ls dist/nacos-mcp_*_linux_amd64.tar.gz | sed -E 's|.*nacos-mcp_([^_]+)_linux_amd64\.tar\.gz|\1|'); \
		SKIP_DOWNLOAD=1 SKIP_UPX=1 VERSION=$$SNAP_VERSION scripts/publish-npm.sh --dry-run
