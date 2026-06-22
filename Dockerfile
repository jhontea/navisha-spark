# Multi-stage build for minimal image size
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o spark ./cmd/spark

# Final stage - minimal image
FROM alpine:latest

# Install ca-certificates for HTTPS calls
RUN apk --no-cache add ca-certificates tzdata

# Set timezone to Asia/Jakarta
ENV TZ=Asia/Jakarta

WORKDIR /app

# Create non-root user for security
RUN addgroup -g 1001 -S spark && \
    adduser -u 1001 -S spark -G spark

# Copy binary and config with correct ownership
COPY --from=builder --chown=spark:spark /app/spark .
COPY --from=builder --chown=spark:spark /app/config/ ./config/

USER spark

# Expose health check port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -q --spider http://localhost:8080/healthz || exit 1

# Run the binary
CMD ["./spark"]