.PHONY: all build test clean install lint fmt coverage

BINARY_NAME=skint
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X main.version=${VERSION}"

all: build

build:
	go build ${LDFLAGS} -o ${BINARY_NAME} .

test:
	go test -v ./...

clean:
	rm -f ${BINARY_NAME}
	go clean

install: build
	mkdir -p ~/.local/bin
	cp ${BINARY_NAME} ~/.local/bin/

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

deps:
	go mod download
	go mod tidy

# Cross-compilation
build-all:
	mkdir -p dist
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o dist/${BINARY_NAME}_darwin_arm64 .
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}_linux_amd64 .
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o dist/${BINARY_NAME}_linux_arm64 .

release: build-all
	cd dist && \
	tar czf ${BINARY_NAME}_${VERSION}_darwin_arm64.tar.gz ${BINARY_NAME}_darwin_arm64 && \
	tar czf ${BINARY_NAME}_${VERSION}_linux_amd64.tar.gz ${BINARY_NAME}_linux_amd64 && \
	tar czf ${BINARY_NAME}_${VERSION}_linux_arm64.tar.gz ${BINARY_NAME}_linux_arm64 && \
