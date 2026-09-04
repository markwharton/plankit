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
