.PHONY: build build-macos install test clean

build:
	go build -o goose-notify ./cmd/goose-notify

build-macos:
	GOOS=darwin GOARCH=arm64 go build -o goose-notify-arm64 ./cmd/goose-notify
	GOOS=darwin GOARCH=amd64 go build -o goose-notify-amd64 ./cmd/goose-notify
	lipo -create -output goose-notify goose-notify-arm64 goose-notify-amd64
	rm goose-notify-arm64 goose-notify-amd64

install: build-macos
	cp goose-notify /usr/local/bin/

test:
	go test -v ./...

clean:
	rm -f goose-notify goose-notify-arm64 goose-notify-amd64
