VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
MODULE     := github.com/timofeevav/gophkeeper

LDFLAGS := -ldflags "\
  -X '$(MODULE)/internal/client/version.Version=$(VERSION)' \
  -X '$(MODULE)/internal/client/version.BuildDate=$(BUILD_DATE)' \
  -X '$(MODULE)/internal/client/version.Commit=$(COMMIT)'"

.PHONY: all build build-server build-client test lint docker-build clean

all: build

build: build-server build-client

build-server:
	go build -o bin/server ./cmd/server

build-client:
	go build $(LDFLAGS) -o bin/gophkeeper ./cmd/client

build-client-all:
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o bin/gophkeeper-linux-amd64   ./cmd/client
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/gophkeeper-windows-amd64.exe ./cmd/client
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o bin/gophkeeper-darwin-amd64  ./cmd/client
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o bin/gophkeeper-darwin-arm64  ./cmd/client

test:
	go test ./... -v -race -count=1

test-coverage:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

docker-build:
	docker build -t gophkeeper-server:$(VERSION) .

clean:
	rm -rf bin/ coverage.out coverage.html

migrate-up:
	migrate -path migrations -database "$(DATABASE_URI)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URI)" down
