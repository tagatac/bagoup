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

build: bin/typedstream-decode bin/bagoup

bin/typedstream-decode: cmd/typedstream-decode/typedstream-decode.m
	mkdir -vp bin
	clang -framework Foundation -o $@ $<

bin/bagoup: $(SRC) $(TEMPLATES) download
	mkdir -vp bin
	go build $(LDFLAGS) -o $@ cmd/bagoup/main.go

.PHONY: deps download from-archive generate test test-pdf clean

deps:
	go get -u -t -v ./...
	go mod tidy -v

download:
	go mod download

example: example-exports/examplegen.go download
	rm -vrf example-exports/messages-export*
	cd example-exports && go run examplegen.go

from-archive:
	BAGOUP_VERSION=$(shell pwd | sed 's/.*bagoup-//g') make build

generate:
	go install github.com/golang/mock/mockgen@latest
	go generate ./...

test: download
	go test -race -coverprofile=$(COVERAGE_FILE) -coverpkg=$(PKGS_TO_COVER) $(PKGS_TO_TEST)
	go tool cover -func=$(COVERAGE_FILE)

test-pdf: download
	rm -vrf test-pdf
	mkdir -vp test-pdf
	cd example-exports && go run examplegen.go ../test-pdf
	cd test-pdf && pdf-diff "../example-exports/messages-export-pdf/Novak Djokovic/iMessage,-,+3815555555555.pdf" "messages-export-pdf/Novak Djokovic/iMessage,-,+3815555555555.pdf" > output.txt
	cat test-pdf/output.txt
	@pdf_diff_result=$$(tail -n 1 test-pdf/output.txt); \
	echo "Output: $$pdf_diff_result"; \
	if [ "$$pdf_diff_result" != "The pages number 1 are the same." ]; then \
		echo "The generated PDF differs from the example"; \
		exit 1; \
	else \
		echo "The generated PDF is the same as the example"; \
	fi

clean:
	rm -vrf \
	bin \
	$(COVERAGE_FILE) \
	test-pdf
