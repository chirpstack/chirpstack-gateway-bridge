.PHONY: build clean test package serve run-compose-test
PKGS := $(shell go list ./... | grep -v /vendor/)
VERSION := $(shell git describe --always)

build:
	@echo "Compiling source"
	@mkdir -p build
	@go build $(GO_EXTRA_BUILD_ARGS) -ldflags "-s -w -X main.version=$(VERSION)" -o build/lora-gateway-bridge cmd/lora-gateway-bridge/main.go

clean:
	@echo "Cleaning up workspace"
	@rm -rf build
	@rm -rf dist
	@rm -rf docs/public

test:
	@echo "Running tests"
	@for pkg in $(PKGS) ; do \
		golint $$pkg ; \
	done
	@go vet $(PKGS)
	@go test -cover -v $(PKGS)

documentation:
	@echo "Building documentation"
	@mkdir -p dist
	@cd docs && hugo
	@cd docs/public/ && tar -pczf ../../dist/lora-gateway-bridge-documentation.tar.gz .

dist:
	@goreleaser

snapshot:
	@goreleaser --snapshot

package-deb: dist
	@cd packaging && TARGET=deb ./package.sh

requirements:
	@go get -u github.com/golang/lint/golint
	@go get -u github.com/kisielk/errcheck
	@go get -u github.com/golang/dep/cmd/dep
	@go get -u github.com/goreleaser/goreleaser
	@dep ensure -v

# shortcuts for development

serve: build
	./build/lora-gateway-bridge

run-compose-test:
	docker-compose run --rm gatewaybridge make test
