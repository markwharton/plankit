BINARY_NAME=pk
VERSION?=dev
BUILD_DIR=dist

# Force pure-Go static binaries; prevents implicit glibc dependency on linux.
export CGO_ENABLED := 0

# Build flags for smaller binaries
LDFLAGS=-s -w -X github.com/markwharton/plankit/internal/version.version=$(VERSION)

.PHONY: all build clean test install fmt lint vet fmtcheck rules-lint vuln cover build-all release release-dry

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

# Lint code: vet + gofmt drift check. Fails if any .go file in a tracked
# package needs formatting. `go list` scopes to real packages so sketch
# dirs (underscore-prefixed) don't produce false positives.
lint: vet fmtcheck

vet:
	go vet ./...

fmtcheck:
	@files=$$(gofmt -l $$(go list -f '{{.Dir}}' ./...)); [ -z "$$files" ] || { echo "gofmt drift:"; echo "$$files"; exit 1; }

# Lint .claude/rules: hidden characters (safety) plus plankit's own writing
# conventions (em dashes, trailing whitespace, hard-wrapped bullets). --strict
# because this repo is the house. Separate from `lint`, which stays pure Go
# and needs no binary; this one runs the freshly built pk.
rules-lint: build
	$(BUILD_DIR)/$(BINARY_NAME) rules --lint --strict

# Scan for known vulnerabilities (Go stdlib + toolchain) against the live
# vuln.go.dev database. Uses the latest govulncheck so detection stays current;
# the database is fetched at run time. Exits non-zero on findings, gating CI.
vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# Coverage of what the next release would ship: per-function coverage for
# every non-test .go file changed since the latest tag, functions below
# 100% only. Codecov judges the release diff after the push; this is the
# same view before it. An empty list means the diff is fully covered.
cover:
	@mkdir -p $(BUILD_DIR)
	@go test -coverprofile=$(BUILD_DIR)/cover.out ./... >/dev/null
	@changed=$$(git diff --name-only $$(git describe --tags --abbrev=0) -- '*.go' | grep -v '_test\.go$$' | sed 's|^|github.com/markwharton/plankit/|'); \
	if [ -z "$$changed" ]; then echo "No .go files changed since $$(git describe --tags --abbrev=0)"; exit 0; fi; \
	echo "Functions below 100% in files changed since $$(git describe --tags --abbrev=0):"; \
	go tool cover -func=$(BUILD_DIR)/cover.out | grep -F "$$changed" | awk '$$3+0 < 100 {print "  " $$0}'

# Release: validate and push to trigger CI build
release:
	pk release

# Dry run: run all release checks without pushing
release-dry:
	pk release --dry-run
