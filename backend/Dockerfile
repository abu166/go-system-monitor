# syntax=docker/dockerfile:1.7
FROM golang:1.22-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git

COPY go.mod go.sum* ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o /bin/server ./cmd/server

FROM alpine:3.20
WORKDIR /app

RUN addgroup -S appgroup && adduser -S appuser -G appgroup && \
    mkdir -p /app/data && chown -R appuser:appgroup /app
COPY --from=builder /bin/server /app/server

USER appuser
ENV BACKEND_PORT=8080
EXPOSE 8080

ENTRYPOINT ["/app/server"]
