COVERAGE_FILE=coverage.out
BAGOUP_VERSION?=$(shell git describe --tags | sed 's/^v//g')
OS=$(shell uname -s)
HW=$(shell uname -m)

SRC=$(shell find . -type f -name '*.go' -not -name '*_test.go' -not -name 'mock_*.go')
TEMPLATES=$(shell find . -type f -name '*.tmpl')
LDFLAGS=-ldflags '-X "main._version=$(BAGOUP_VERSION) $(OS)/$(HW)"'

PKGS=$(shell go list ./... | grep --invert-match '/mock_' | tr '\n' ' ')
EXCLUDE_PKGS=\
	github.com/tagatac/bagoup/v2/example-exports \
	github.com/tagatac/bagoup/v2/exectest
PKGS_TO_TEST=$(filter-out $(EXCLUDE_PKGS),$(PKGS))
PKGS_TO_COVER=$(shell echo "$(PKGS_TO_TEST)" | tr ' ' ',')

EXAMPLE_EXPORT_FILE='Novak Djokovic/iMessage,-,+3815555555555'
TXT_FILE=messages-export/$(EXAMPLE_EXPORT_FILE).txt
PDF_FILE=messages-export-pdf/$(EXAMPLE_EXPORT_FILE).pdf
PDF_FILE_WKHTML=messages-export-wkhtmltopdf/$(EXAMPLE_EXPORT_FILE).pdf
PDFINFO_IGNORE_CMD=grep -Ev 'Creator|CreationDate|File size'
EXAMPLE_EXPORTS_DIR=example-exports
TEST_EXPORTS_DIR=test-exports

build: bin/typedstream-decode bin/bagoup

bin/typedstream-decode: cmd/typedstream-decode/typedstream-decode.m
	mkdir -vp bin
	clang -framework Foundation -o $@ $<

bin/bagoup: $(SRC) $(TEMPLATES) download
	mkdir -vp bin
	go build $(LDFLAGS) -o $@ cmd/bagoup/main.go

.PHONY: deps download from-archive generate test test-exports clean

deps:
	go get -u -t -v ./...
	go mod tidy -v

download:
	go mod download

example: example-exports/examplegen.go download
	rm -vrf example-exports/messages-export*
	cd example-exports && go run $(LDFLAGS) examplegen.go

from-archive:
	BAGOUP_VERSION=$(shell pwd | sed 's/.*bagoup-//g') make build

generate:
	go install github.com/golang/mock/mockgen@latest
	go generate ./...

test: download
	go test -race -coverprofile=$(COVERAGE_FILE) -coverpkg=$(PKGS_TO_COVER) $(PKGS_TO_TEST)
	go tool cover -func=$(COVERAGE_FILE)

test-exports: download
	rm -vrf $(TEST_EXPORTS_DIR)
	mkdir -vp $(TEST_EXPORTS_DIR)
	cd $(EXAMPLE_EXPORTS_DIR) && go run $(LDFLAGS) examplegen.go ../$(TEST_EXPORTS_DIR)
	diff $(EXAMPLE_EXPORTS_DIR)/$(TXT_FILE) $(TEST_EXPORTS_DIR)/$(TXT_FILE)
	bash -c "diff <(pdfinfo $(EXAMPLE_EXPORTS_DIR)/$(PDF_FILE) | $(PDFINFO_IGNORE_CMD)) <(pdfinfo $(TEST_EXPORTS_DIR)/$(PDF_FILE) | $(PDFINFO_IGNORE_CMD))"
	diff-pdf -v $(EXAMPLE_EXPORTS_DIR)/$(PDF_FILE) $(TEST_EXPORTS_DIR)/$(PDF_FILE)
	bash -c "diff <(pdfinfo $(EXAMPLE_EXPORTS_DIR)/$(PDF_FILE_WKHTML) | $(PDFINFO_IGNORE_CMD)) <(pdfinfo $(TEST_EXPORTS_DIR)/$(PDF_FILE_WKHTML) | $(PDFINFO_IGNORE_CMD))"
	diff-pdf -v $(EXAMPLE_EXPORTS_DIR)/$(PDF_FILE_WKHTML) $(TEST_EXPORTS_DIR)/$(PDF_FILE_WKHTML)

clean:
	rm -vrf \
	bin \
	$(COVERAGE_FILE) \
	$(TEST_EXPORTS_DIR)
