# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /eval-mcp ./cmd/mcp

# Runtime stage - minimal image for MCP stdio
FROM alpine:3.22

# MCP communicates over stdio; no shell needed for stdio transport
COPY --from=builder /eval-mcp /eval-mcp

# AWS credentials and config passed at runtime via -e or --env-file
ENV AWS_REGION=us-east-1

ENTRYPOINT ["/eval-mcp"]
