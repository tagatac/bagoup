BINARY=bagoup
ZIPFILE=bagoup-darwin-x86_64.zip

$(BINARY): main.go opsys/opsys.go chatdb/chatdb.go
	go build -o $@ $<

.PHONY: generate test zip clean

generate:
	go get -u github.com/golang/mock/mockgen
	go generate ./...

test:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

zip: bagoup
	zip $(ZIPFILE) bagoup

clean:
	rm -vf $(BINARY) coverage.out $(ZIPFILE)
