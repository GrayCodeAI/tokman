.PHONY: build build-small build-all build-simd build-tiny test test-cover lint typecheck vet fmt clean benchmark check

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Go 1.26+ with SIMD support
GO ?= ~/sdk/go1.26.0/bin/go
GOEXPERIMENT ?= simd

# T30: Aggressive optimization flags for smaller binary
LDFLAGS := -s -w -X github.com/GrayCodeAI/tokman/internal/commands.Version=$(VERSION)

# Standard build (stripped symbols)
build:
	$(GO) build -ldflags="$(LDFLAGS)" -o bin/tokman ./cmd/tokman

# T106: Optimized small binary (strip + compress)
build-small:
	$(GO) build -ldflags="$(LDFLAGS)" -trimpath -o bin/tokman ./cmd/tokman
	@echo "Binary size: $$(du -h bin/tokman | cut -f1)"

# T30: Tiny binary with maximum optimization
build-tiny:
	CGO_ENABLED=0 $(GO) build -ldflags="$(LDFLAGS) -extldflags '-static'" -trimpath -tags netgo,osusergo -o bin/tokman-tiny ./cmd/tokman
	@echo "Tiny binary size: $$(du -h bin/tokman-tiny | cut -f1)"
	@if command -v upx >/dev/null 2>&1; then \
		upx --best bin/tokman-tiny -o bin/tokman-upx 2>/dev/null || true; \
		echo "UPX compressed size: $$(du -h bin/tokman-upx 2>/dev/null | cut -f1 || echo 'N/A')"; \
	fi

# SIMD-optimized build (requires Go 1.26+)
build-simd:
	GOEXPERIMENT=$(GOEXPERIMENT) $(GO) build -ldflags="$(LDFLAGS)" -trimpath -o bin/tokman-simd ./cmd/tokman
	@echo "SIMD binary size: $$(du -h bin/tokman-simd | cut -f1)"

# Multi-platform build with SIMD
build-all:
	GOEXPERIMENT=$(GOEXPERIMENT) GOOS=linux GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -trimpath -o bin/tokman-linux-amd64 ./cmd/tokman
	GOEXPERIMENT=$(GOEXPERIMENT) GOOS=linux GOARCH=arm64 $(GO) build -ldflags="$(LDFLAGS)" -trimpath -o bin/tokman-linux-arm64 ./cmd/tokman
	GOEXPERIMENT=$(GOEXPERIMENT) GOOS=darwin GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -trimpath -o bin/tokman-darwin-amd64 ./cmd/tokman
	GOEXPERIMENT=$(GOEXPERIMENT) GOOS=darwin GOARCH=arm64 $(GO) build -ldflags="$(LDFLAGS)" -trimpath -o bin/tokman-darwin-arm64 ./cmd/tokman
	GOEXPERIMENT=$(GOEXPERIMENT) GOOS=windows GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -trimpath -o bin/tokman-windows-amd64.exe ./cmd/tokman

test:
	GOEXPERIMENT=$(GOEXPERIMENT) $(GO) test -race -count=1 ./...

test-short:
	GOEXPERIMENT=$(GOEXPERIMENT) $(GO) test -short -count=1 ./...

test-cover:
	GOEXPERIMENT=$(GOEXPERIMENT) $(GO) test -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

typecheck:
	$(GO) vet ./...

vet:
	$(GO) vet ./...

fmt:
	gofmt -s -w .
	goimports -w .

clean:
	rm -rf bin/ coverage.out coverage.html

benchmark:
	GOEXPERIMENT=$(GOEXPERIMENT) $(GO) test -bench=. -benchmem ./...

# Run all checks
check: fmt vet typecheck lint test

# Quick check (skip slow tests)
check-quick: fmt vet typecheck lint test-short

# CI check (what CI runs)
ci: test lint benchmark