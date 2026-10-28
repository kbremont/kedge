SERVER  := server
BINARY  := bin/531
VERSION := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)

.PHONY: all generate build test run fmt vet lint check icons docker app clean

all: check build

## generate: regenerate templ templates and sqlc queries
## The tools are pinned in server/go.mod and run with go tool.
generate:
	cd $(SERVER) && go tool templ generate
	cd $(SERVER) && go tool sqlc generate

## icons: redraw the PWA icons
icons:
	cd $(SERVER) && go run ./tools/genicons

## build: compile the server to server/bin/531
build: generate
	cd $(SERVER) && go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o $(BINARY) ./cmd/server

## test: run the server test suite
test:
	cd $(SERVER) && go test ./...

fmt:
	cd $(SERVER) && gofmt -w cmd internal tools

vet:
	cd $(SERVER) && go vet ./...

lint:
	cd $(SERVER) && go tool staticcheck ./...

## check: fmt, vet, staticcheck and test
check: fmt vet lint test

## run: start the server locally against server/data
run: generate
	cd $(SERVER) && AUTH_PASSWORD=$${AUTH_PASSWORD:-dev} DATA_DIR=./data ADDR=:3000 go run ./cmd/server

## docker: build a linux/arm64 image for the Pi
docker:
	docker buildx build --platform linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		-t 531-tracker:$(VERSION) $(SERVER)

## app: start the Expo dev server for the iOS app
app:
	cd mobile && npx expo start

clean:
	rm -rf $(SERVER)/bin
