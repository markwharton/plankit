BINARY_NAME=pk
VERSION?=dev
BUILD_DIR=dist

# Force pure-Go static binaries; prevents implicit glibc dependency on linux.
export CGO_ENABLED := 0

# Build flags for smaller binaries
LDFLAGS=-s -w -X github.com/markwharton/plankit/internal/version.version=$(VERSION)

.PHONY: all build clean test install fmt lint build-all release release-dry

all: build

# Build for current platform
build:
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/pk

# Build for all platforms
build-all: clean
	mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/pk
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/pk
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/pk
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/pk
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/pk

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

# Run tests (race detector requires cgo)
test:
	CGO_ENABLED=1 go test -v -race ./...

# Install to GOPATH/bin
install:
	go install -ldflags "$(LDFLAGS)" ./cmd/pk

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	go vet ./...

# Release: validate and push to trigger CI build
release:
	pk release

# Dry run: run all release checks without pushing
release-dry:
	pk release --dry-run
