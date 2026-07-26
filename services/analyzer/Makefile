.PHONY: build clean test

build:
	GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o build/bootstrap ./cmd/analyzer
	cd build && zip analyzer.zip bootstrap

clean:
	rm -rf build/

test:
	go test -v -race ./...
