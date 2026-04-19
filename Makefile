COVERAGE_FILE=coverage.out
BAGOUP_VERSION?=$(shell git describe --tags | sed 's/^v//g')
OS=$(shell uname -s)
HW=$(shell uname -m)

SRC=$(shell find . -type f -name '*.go' -not -name '*_test.go' -not -name 'mock_*.go')
TEMPLATES=$(shell find . -type f -wholename './opsys/templates/*.tmpl')
LDFLAGS=-ldflags '-X "main._version=$(BAGOUP_VERSION) $(OS)/$(HW)"'

PKGS=$(shell go list ./... | grep --invert-match '/mock_' | tr '\n' ' ')
EXCLUDE_PKGS=\
	github.com/tagatac/bagoup/v2/example-exports \
	github.com/tagatac/bagoup/v2/exectest
PKGS_TO_TEST=$(filter-out $(EXCLUDE_PKGS),$(PKGS))
PKGS_TO_COVER=$(shell echo "$(PKGS_TO_TEST)" | tr ' ' ',')

EXAMPLE_EXPORTS_DIR=example-exports/$(OS)
TEST_EXPORTS_DIR=test-exports

build: bin/bagoup

bin/bagoup: $(SRC) $(TEMPLATES) download
	mkdir -vp bin
	go build $(LDFLAGS) -o $@ cmd/bagoup/main.go

.PHONY: deps download from-archive generate vet test test-exports clean

deps:
	go get -u -t -v ./...
	go mod tidy -v

download:
	go mod download

example: example-exports/examplegen.go download
	rm -vrf $(EXAMPLE_EXPORTS_DIR)
	cd example-exports && go run examplegen.go $(OS)

from-archive:
	BAGOUP_VERSION=$(shell pwd | sed 's/.*bagoup-//g') make build

generate:
	go install go.uber.org/mock/mockgen@latest
	go generate ./...

vet:
	go vet ./...

test: download
	go test -race -coverprofile=$(COVERAGE_FILE) -coverpkg=$(PKGS_TO_COVER) $(PKGS_TO_TEST)
	go tool cover -func=$(COVERAGE_FILE)

test-exports: download
	bash scripts/test-exports.sh

clean:
	rm -vrf \
	bin \
	$(COVERAGE_FILE) \
	$(TEST_EXPORTS_DIR)
	go clean -testcache
