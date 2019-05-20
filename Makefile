.PHONY: build clean test package serve run-compose-test
PKGS := $(shell go list ./... | grep -v /vendor/)
VERSION := $(shell git describe --always |sed -e "s/^v//")

build:
	@echo "Compiling source"
	@mkdir -p build
	go build $(GO_EXTRA_BUILD_ARGS) -ldflags "-s -w -X main.version=$(VERSION)" -o build/lora-gateway-bridge cmd/lora-gateway-bridge/main.go

clean:
	@echo "Cleaning up workspace"
	@rm -rf build
	@rm -rf dist
	@rm -rf docs/public

test:
	@echo "Running tests"
	@rm -f coverage.out
	@for pkg in $(PKGS) ; do \
		golint $$pkg ; \
	done
	@go vet $(PKGS)
	@go test -cover -v $(PKGS) -coverprofile coverage.out

dist:
	@goreleaser
	mkdir -p dist/upload/tar
	mkdir -p dist/upload/deb
	mv dist/*.tar.gz dist/upload/tar
	mv dist/*.deb dist/upload/deb

snapshot:
	@goreleaser --snapshot

dev-requirements:
	go install golang.org/x/lint/golint
	go install github.com/goreleaser/goreleaser
	go install github.com/goreleaser/nfpm

# shortcuts for development

serve: build
	./build/lora-gateway-bridge

run-compose-test:
	docker-compose run --rm gatewaybridge make test
