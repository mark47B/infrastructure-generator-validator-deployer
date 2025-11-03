FROM golang:1.24.6-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o orchestrator ./app/cmd/main.go


FROM hashicorp/terraform:1.9.8

USER root
RUN apk add --no-cache ca-certificates tzdata bash

RUN adduser -D -s /bin/sh appuser

WORKDIR /app
COPY --from=builder /app/orchestrator .

RUN mkdir -p /app/deployments && chown -R appuser:appuser /app

USER appuser

EXPOSE 8080

ENV SERVER_HOST=0.0.0.0
ENV SERVER_PORT=8080
ENV STORAGE_BASE_PATH=/app/deployments

ENTRYPOINT ["/app/orchestrator"]
