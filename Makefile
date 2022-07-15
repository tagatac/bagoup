COVERAGE_FILE=coverage.out
VERSION=$(shell git describe --tags | sed 's/^v//g')
ZIPFILE="bagoup-$(VERSION)-$(shell uname -s)-$(shell uname -m).zip"

build: bagoup

bagoup: main.go opsys/opsys.go opsys/outfile.go opsys/templates/* chatdb/chatdb.go pathtools/pathtools.go download
	go build -o $@ $<

.PHONY: deps download generate test zip clean codecov

deps:
	go get -u -v ./...
	go mod tidy -v
	go get -u golang.org/x/tools/cover

download:
	go mod download

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
