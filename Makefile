.PHONY: build install clean test

BINARY_NAME=docker-dev-swap
MAIN_PATH=./main.go

build:
	go build -o ${BINARY_NAME} ${MAIN_PATH}

install:
	go install ${MAIN_PATH}

clean:
	go clean
	rm -f ${BINARY_NAME}

test:
	go test -v ./...

run:
	go run ${MAIN_PATH} -config config.yaml

lint:
	golangci-lint run

deps:
	go mod download
	go mod tidy