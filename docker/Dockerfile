# TokMan Docker Image
# Multi-stage build for minimal image size

# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X github.com/GrayCodeAI/tokman/internal/commands.Version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" -o tokman ./cmd/tokman

# Runtime stage
FROM alpine:3.19

LABEL maintainer="GrayCodeAI <https://github.com/GrayCodeAI>"
LABEL org.opencontainers.image.source="https://github.com/GrayCodeAI/tokman"
LABEL org.opencontainers.image.description="Token-aware CLI proxy for LLM interactions"
LABEL org.opencontainers.image.licenses="MIT"

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    git \
    curl \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN adduser -D -u 1000 tokman

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/tokman /usr/local/bin/tokman

# Copy completions and templates
COPY completions/ /usr/share/tokman/completions/
COPY templates/ /usr/share/tokman/templates/

# Set permissions
RUN chmod +x /usr/local/bin/tokman && \
    chown -R tokman:tokman /app

# Switch to non-root user
USER tokman

# Set up data directory
RUN mkdir -p /home/tokman/.config/tokman /home/tokman/.local/share/tokman

# Environment
ENV TOKMAN_DATA_DIR=/home/tokman/.local/share/tokman
ENV TOKMAN_CONFIG_DIR=/home/tokman/.config/tokman

# Default entrypoint
ENTRYPOINT ["tokman"]
CMD ["--help"]
