SERVER  := server
BINARY  := bin/531
VERSION := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)

.PHONY: all generate golden build test run demo fmt vet lint check icons docker app clean

all: check build

FIXTURES := mobile/src/api/fixtures

## generate: regenerate templ templates, sqlc queries, and both ends of the
## API from its spec in server/api/openapi.yaml: the server, and the app's
## types and fixtures. The Go tools are pinned in server/go.mod and run with
## go tool; openapi-typescript is pinned in mobile/package.json, so run
## npm ci in mobile/ first. The fixtures are the golden responses; refresh
## those with make golden first.
generate:
	cd $(SERVER) && go tool templ generate
	cd $(SERVER) && go tool sqlc generate
	cd $(SERVER)/internal/api && go tool oapi-codegen -config oapi-codegen.yaml ../../api/openapi.yaml
	cd mobile && npx --no -- openapi-typescript ../$(SERVER)/api/openapi.yaml -o src/api/schema.gen.ts \
		--root-types --root-types-no-schema-prefix --root-types-keep-casing
	rm -rf $(FIXTURES)
	mkdir -p $(FIXTURES)
	cp $(SERVER)/internal/api/testdata/*.json $(FIXTURES)/

## golden: rewrite the API's golden responses in server/internal/api/testdata
golden:
	cd $(SERVER) && go test ./internal/api -update

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

## demo: start the server against server/data-demo, filled with a realistic
## demo block the first time
demo: generate
	cd $(SERVER) && AUTH_PASSWORD=$${AUTH_PASSWORD:-dev} DATA_DIR=./data-demo ADDR=:3000 go run ./cmd/server -seed-demo

## docker: build a linux/arm64 image for the Pi
docker:
	docker buildx build --platform linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		-t kedge:$(VERSION) $(SERVER)

## app: start the Expo dev server for the iOS app
app:
	cd mobile && npx expo start

clean:
	rm -rf $(SERVER)/bin
