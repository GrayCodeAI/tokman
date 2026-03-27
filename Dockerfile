# syntax=docker/dockerfile:1
# Task #143: Docker multi-stage build optimization.
# Stage 1: builder — compiles the binary with full Go toolchain.
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /build

# Cache dependency downloads separately from source compilation.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build a statically-linked binary.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" \
    -o /tokman ./cmd/tokman

# ─────────────────────────────────────────────────────────────────────────────
# Stage 2: final — minimal runtime image (< 15 MB).
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /tokman /tokman

# Expose no ports — tokman is a CLI/library, not a server by default.
ENTRYPOINT ["/tokman"]
CMD ["--help"]
