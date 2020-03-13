build: bagoup

bagoup: main.go chatdb/db.go
	go build -o bagoup main.go

.PHONY: test clean

test:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

clean:
	rm -vf bagoup coverage.out
