COVERAGE_FILE=coverage.out
ZIPFILE="bagoup-$(shell uname -s)-$(shell uname -m).zip"
GOHEIF_VENDOR_DIR=vendor/github.com/adrium/goheif
LIBDE265_VENDOR_DIR=$(GOHEIF_VENDOR_DIR)/libde265

build: bagoup

bagoup: main.go opsys/opsys.go opsys/outfile.go opsys/templates/* chatdb/chatdb.go pathtools/pathtools.go vendor
	go build -o $@ $<

vendor: go.mod go.sum
	go mod vendor -v
	rm -vrf $(LIBDE265_VENDOR_DIR)
	@echo "Copy files pruned by `go mod vendor` (see https://github.com/golang/go/issues/26366). Sudo permissions will be required"
	cp -vR $(shell go env GOPATH)/pkg/mod/github.com/adrium/goheif@v0.0.0-20210309200126-b184a7b446fa/libde265 $(GOHEIF_VENDOR_DIR)

.PHONY: deps generate test zip clean codecov

deps:
	go get -u -v ./...
	go mod tidy -v
	go get -u golang.org/x/tools/cover

generate: clean
	go get -u github.com/golang/mock/mockgen
	go generate ./...

test: vendor
	go test -race -coverprofile=$(COVERAGE_FILE) ./...
	go tool cover -func=$(COVERAGE_FILE)

zip: build
	zip $(ZIPFILE) bagoup

clean:
	sudo rm -vrf $(LIBDE265_VENDOR_DIR)
	rm -vrf bagoup \
	vendor \
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
