# syntax=docker/dockerfile:1.7

FROM golang:1.27.3-alpine AS build

RUN apk add --no-cache make

WORKDIR /src
ENV CGO_ENABLED=0 GOTOOLCHAIN=local GOFLAGS=-mod=readonly

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    make swagger && \
    go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.22

RUN addgroup -S -g 10001 app && adduser -S -D -H -u 10001 -G app app

COPY --from=build /out/api /app/api

USER app:app
EXPOSE 8080

HEALTHCHECK --interval=1s --timeout=3s --start-period=20s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/readyz || exit 1
ENTRYPOINT ["/app/api"]
