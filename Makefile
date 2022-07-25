COVERAGE_FILE=coverage.out
BAGOUP_VERSION?=$(shell git describe --tags | sed 's/^v//g')
OS=$(shell uname -s)
HW=$(shell uname -m)
ZIPFILE="bagoup-$(BAGOUP_VERSION)-$(OS)-$(HW).zip"

SRC=$(shell find . -type f -name '*.go' -not -name '*_test.go' -not -name 'mock_*.go')
TEMPLATES=$(shell find . -type f -name '*.tmpl')
LDFLAGS=-ldflags '-X "main._version=$(BAGOUP_VERSION) $(OS)/$(HW)"'

build: bagoup

bagoup: $(SRC) $(TEMPLATES) download
	go build $(LDFLAGS) -o $@ .

.PHONY: deps download from-archive generate test zip clean codecov

deps:
	go get -u -v ./...
	go mod tidy -v
	go get -u golang.org/x/tools/cover

download:
	go mod download

from-archive:
	BAGOUP_VERSION=$(shell pwd | sed 's/.*bagoup-//g') make bagoup

generate: clean
	go get -u github.com/golang/mock/mockgen
	go generate ./...
	make deps

test: download
	go test -race -coverprofile=$(COVERAGE_FILE) ./...
	go tool cover -func=$(COVERAGE_FILE)

zip: build
	zip $(ZIPFILE) bagoup

clean:
	rm -vrf bagoup \
	$(COVERAGE_FILE) \
	$(ZIPFILE)

codecov:
	curl https://keybase.io/codecovsecurity/pgp_keys.asc | gpg --import
	curl -Os https://uploader.codecov.io/latest/linux/codecov
	curl -Os https://uploader.codecov.io/latest/linux/codecov.SHA256SUM
	curl -Os https://uploader.codecov.io/latest/linux/codecov.SHA256SUM.sig
	gpg --verify codecov.SHA256SUM.sig codecov.SHA256SUM
	shasum -a 256 -c codecov.SHA256SUM
	chmod +x codecov
	./codecov
