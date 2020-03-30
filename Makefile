COVERAGE_FILE=coverage.out
ZIPFILE=bagoup-darwin-x86_64.zip

build: bagoup

bagoup: main.go opsys/opsys.go chatdb/chatdb.go vendor
	go build -o $@ $<

vendor: go.mod go.sum
	go mod vendor -v

.PHONY: deps generate test zip clean

deps:
	go mod tidy -v
	go get -u -v ./...
	go get -u golang.org/x/tools/cover

generate:
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
