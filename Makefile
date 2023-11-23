COVERAGE_FILE=coverage.out
BAGOUP_VERSION?=$(shell git describe --tags | sed 's/^v//g')
OS=$(shell uname -s)
HW=$(shell uname -m)
ZIPFILE="bagoup-$(BAGOUP_VERSION)-$(OS)-$(HW).zip"

SRC=$(shell find . -type f -name '*.go' -not -name '*_test.go' -not -name 'mock_*.go')
TEMPLATES=$(shell find . -type f -name '*.tmpl')
LDFLAGS=-ldflags '-X "main._version=$(BAGOUP_VERSION) $(OS)/$(HW)"'

build: bin/typedstream-decode bin/bagoup

bin/typedstream-decode:
	make -C cmd/typedstream-decode
	mkdir -vp bin
	cp -vf cmd/typedstream-decode/typedstream-decode bin

bin/bagoup: $(SRC) $(TEMPLATES) download
	mkdir -vp bin
	go build $(LDFLAGS) -o $@ cmd/bagoup/main.go

.PHONY: deps download from-archive generate test zip clean

deps:
	go get -u -v ./...
	go mod tidy -v
	go get -u golang.org/x/tools/cover

download:
	go mod download

example: example-exports/examplegen.go download
	rm -vrf example-exports/messages-export*
	cd example-exports && go run examplegen.go

from-archive:
	BAGOUP_VERSION=$(shell pwd | sed 's/.*bagoup-//g') make build

generate: clean deps
	go install github.com/golang/mock/mockgen@latest
	go generate ./...

test: download
	go test -race -coverprofile=$(COVERAGE_FILE) ./...
	go tool cover -func=$(COVERAGE_FILE)

zip: build
	zip --recurse-paths $(ZIPFILE) bin

clean:
	rm -vrf \
	bin \
	$(COVERAGE_FILE) \
	$(ZIPFILE)
	make -C cmd/typedstream-decode clean
