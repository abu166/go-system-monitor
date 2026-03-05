# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/server ./cmd/server

FROM alpine:3.20
WORKDIR /app

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
COPY --from=builder /bin/server /app/server
RUN mkdir -p /app/data && chown -R appuser:appgroup /app

USER appuser
ENV BACKEND_PORT=8080
EXPOSE 8080

ENTRYPOINT ["/app/server"]
