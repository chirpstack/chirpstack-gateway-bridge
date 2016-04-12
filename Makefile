PKGS := $(shell go list ./... | grep -v /vendor/)
VERSION := $(shell git describe --always)

build:
	@echo "Compiling source"
	@mkdir -p bin
	@GOBIN="$(CURDIR)/bin" go install -ldflags "-X main.version=$(VERSION)" $(PKGS)

clean:
	@echo "Cleaning up workspace"
	@rm -rf bin

test:
	@echo "Running tests"
	@for pkg in $(PKGS) ; do \
		golint $$pkg ; \
	done
	@go vet $(PKGS)
	@go test -cover -v $(PKGS)

package: clean build
	@echo "Creating package"
	@mkdir -p builds/$(VERSION)
	@cp bin/* builds/$(VERSION)
	@cd builds/$(VERSION)/ && tar -pczf ../lora_semtech_bridge_$(VERSION)_linux_amd64.tar.gz .
	@rm -rf builds/$(VERSION)

# shortcuts for development

serve: build
	./bin/semtech-bridge

run-compose-test:
	docker-compose run --rm semtechbridge make test
