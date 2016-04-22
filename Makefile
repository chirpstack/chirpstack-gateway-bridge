PKGS := $(shell go list ./... | grep -v /vendor/)
VERSION := $(shell git describe --always)
GOOS ?= linux
GOARCH ?= amd64

build:
	@echo "Compiling source for $(GOOS) $(GOARCH)"
	@mkdir -p build
	@GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "-X main.version=$(VERSION)" -o build/semtech-bridge$(BINEXT) cmd/semtech-bridge/main.go

clean:
	@echo "Cleaning up workspace"
	@rm -rf build
	@rm -rf dist/$(VERSION)

test:
	@echo "Running tests"
	@for pkg in $(PKGS) ; do \
		golint $$pkg ; \
	done
	@go vet $(PKGS)
	@go test -cover -v $(PKGS)

package: clean build
	@echo "Creating package for $(GOOS) $(GOARCH)"
	@mkdir -p dist/$(VERSION)
	@cp build/* dist/$(VERSION)
	@cd dist/$(VERSION)/ && tar -pczf ../lora_semtech_bridge_$(VERSION)_$(GOOS)_$(GOARCH).tar.gz .
	@rm -rf dist/$(VERSION)

# shortcuts for development

serve: build
	./build/semtech-bridge

run-compose-test:
	docker-compose run --rm semtechbridge make test
