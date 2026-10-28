# The build stage needs a Go toolchain; the result needs nothing at all.
# templ and sqlc output is committed, so neither tool is required here.
#
# Pinned to $BUILDPLATFORM so multi-arch builds cross-compile natively on the
# runner instead of emulating the arm64 build stage under QEMU.
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build

WORKDIR /src

# No separate go mod download step: go.mod also pins the dev tools (sqlc and
# friends), whose dependencies run to over a gigabyte. go build fetches only
# the modules the binary imports, and the cache mounts keep them, and the
# build cache, between local builds.
COPY . .

ARG VERSION=dev
ARG TARGETOS
ARG TARGETARCH
# CGO stays off: modernc.org/sqlite is pure Go, which is what makes a static
# binary and a painless cross-compile to arm64 possible.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
        -trimpath \
        -ldflags "-s -w -X main.version=${VERSION}" \
        -o /out/server ./cmd/server

FROM scratch

COPY --from=build /out/server /server

ENV DATA_DIR=/data \
    ADDR=:3000

EXPOSE 3000
VOLUME ["/data"]

ENTRYPOINT ["/server"]
