build: bagoup

bagoup: main.go chatdb/db.go
	go build -o bagoup main.go

.PHONY: test clean

test:
	go test ./...

clean:
	rm -vf bagoup
