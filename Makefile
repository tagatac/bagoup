COVERAGE_FILE=coverage.out
ZIPFILE="bagoup-$(shell uname -s)-$(shell uname -m).zip"

build: bagoup

bagoup: main.go opsys/opsys.go opsys/outfile.go opsys/*.tmpl chatdb/chatdb.go vendor
	go build -o $@ $<

vendor: go.mod go.sum
	go mod vendor -v

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
