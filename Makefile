# VERSION stamps the binaries: set it from the tag at release (release.yml
# passes the tag without its v), leave it empty for a development build,
# which then reports dev+<commit>.
VERSION ?=
LDFLAGS = -s -w $(if $(VERSION),-X github.com/markwharton/plankit/internal/version.stamped=$(VERSION))

.PHONY: build test docs site site-preview vet fmt

build: docs
	go build -ldflags "$(LDFLAGS)" -o pk ./cmd/pk

docs:
	cd tools/docgen && go run . -skills ../../skills -out ../../internal/help/data

site: build
	cd tools/docgen && go run . -skills ../../skills -out ../../internal/help/data -site ../../site/dist -root ../.. -pk ../../pk

# Local preview: links end in .html for a plain static server, and release
# notes whose tag does not exist yet are rendered too.
site-preview: build
	cd tools/docgen && go run . -skills ../../skills -out ../../internal/help/data -site ../../site/dist -root ../.. -pk ../../pk -notes all -links html

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
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/pk-$$t$$ext ./cmd/pk; \
	done

bin-local: docs
	go build -ldflags "$(LDFLAGS)" -o bin/pk-$$(go env GOOS)-$$(go env GOARCH) ./cmd/pk
