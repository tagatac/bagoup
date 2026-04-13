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

EXAMPLE_EXPORT_FILE='Novak Djokovic/iMessage;-;+3815555555555'
TXT_FILE=messages-export/$(EXAMPLE_EXPORT_FILE).txt
PDF_FILE=messages-export-pdf/$(EXAMPLE_EXPORT_FILE).pdf
PDF_FILE_WKHTML=messages-export-wkhtmltopdf/$(EXAMPLE_EXPORT_FILE).pdf
PDFINFO_IGNORE_CMD=grep -Ev 'Creator|CreationDate|File size|Producer'
EXAMPLE_EXPORTS_DIR=example-exports/$(OS)
TEST_EXPORTS_DIR=test-exports

build: bin/typedstream-decode bin/bagoup

bin/typedstream-decode: cmd/typedstream-decode/typedstream-decode.m
	mkdir -vp bin
	clang -framework Foundation -o $@ $<

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

define compare-pdf
	bash -c " \
		pdfinfo $(EXAMPLE_EXPORTS_DIR)/$(1) | $(PDFINFO_IGNORE_CMD) > $(TEST_EXPORTS_DIR)/$(1).expected.info && \
		pdfinfo $(TEST_EXPORTS_DIR)/$(1) | $(PDFINFO_IGNORE_CMD) > $(TEST_EXPORTS_DIR)/$(1).actual.info && \
		diff $(TEST_EXPORTS_DIR)/$(1).expected.info $(TEST_EXPORTS_DIR)/$(1).actual.info"
	magick compare -verbose -metric SSIM \
	  \( -density 300 $(EXAMPLE_EXPORTS_DIR)/$(1) -background white -alpha remove \) \
	  \( -density 300 $(TEST_EXPORTS_DIR)/$(1) -background white -alpha remove \) \
	  null: 2>&1 \
		| tee /dev/stderr \
		| grep -i "all" \
		| awk 'BEGIN { found=0 } { found=1; val=$$2 } END { \
			if (!found) exit 1; \
	    if (val > 0.5) { exit (val >= 0.999 ? 0 : 1) } \
	    else { exit (val <= 0.001 ? 0 : 1) } \
		}'
endef

test-exports: download
	rm -vrf $(TEST_EXPORTS_DIR)
	mkdir -vp $(TEST_EXPORTS_DIR)
	cd example-exports && go run examplegen.go ../$(TEST_EXPORTS_DIR)
	diff $(EXAMPLE_EXPORTS_DIR)/$(TXT_FILE) $(TEST_EXPORTS_DIR)/$(TXT_FILE)
	$(call compare-pdf,$(PDF_FILE))
	$(call compare-pdf,$(PDF_FILE_WKHTML))

clean:
	rm -vrf \
	bin \
	$(COVERAGE_FILE) \
	$(TEST_EXPORTS_DIR)
	go clean -testcache
