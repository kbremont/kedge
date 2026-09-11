BINARY  := bin/531
VERSION := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)

.PHONY: all generate build test run fmt vet check icons docker clean

all: check build

## generate: regenerate templ templates and sqlc queries
generate:
	templ generate
	sqlc generate

## icons: redraw the PWA icons
icons:
	go run ./tools/genicons

## build: compile the server
build: generate
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o $(BINARY) ./cmd/server

## test: run the test suite
test:
	go test ./...

fmt:
	gofmt -w cmd internal tools

vet:
	go vet ./...

## check: fmt, vet and test
check: fmt vet test

## run: start the server locally against ./data
run: generate
	AUTH_PASSWORD=$${AUTH_PASSWORD:-dev} DATA_DIR=./data ADDR=:3000 go run ./cmd/server

## docker: build a linux/arm64 image for the Pi
docker:
	docker buildx build --platform linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		-t 531-tracker:$(VERSION) .

clean:
	rm -rf bin
