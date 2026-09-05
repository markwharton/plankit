.PHONY: build test docs vet fmt

build: docs
	go build -o pk ./cmd/pk

docs:
	cd tools/docgen && go run . -skills ../../skills -out ../../internal/help/data

test:
	go vet ./...
	go test ./...
	cd tools/docgen && go vet ./...

fmt:
	gofmt -w cmd internal
	cd tools/docgen && gofmt -w .

# Cross-compile the platform binaries the bin/ shims dispatch to. Release
# assets in layer 6; bin-local is enough for --plugin-dir testing.
TRIPLES := darwin-arm64 darwin-amd64 linux-amd64 linux-arm64 windows-amd64 windows-arm64

.PHONY: dist bin-local
dist: docs
	@for t in $(TRIPLES); do \
		os=$${t%-*}; arch=$${t#*-}; ext=""; \
		if [ "$$os" = windows ]; then ext=.exe; fi; \
		echo "  bin/pk-$$t$$ext"; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o bin/pk-$$t$$ext ./cmd/pk; \
	done

bin-local: docs
	go build -o bin/pk-$$(go env GOOS)-$$(go env GOARCH) ./cmd/pk
