build: bagoup

bagoup: main.go chatdb/db.go
	go build -o bagoup main.go

.PHONY: generate test clean

generate:
	go get -u github.com/golang/mock/mockgen
	go generate ./...

test:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

clean:
	rm -vf bagoup coverage.out
