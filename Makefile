.PHONY: build build-small build-all test test-cover lint typecheck vet fmt clean benchmark check

# Standard build (stripped symbols)
build:
	go build -ldflags="-s -w" -o bin/tokman ./cmd/tokman

# T106: Optimized small binary (strip + compress)
build-small:
	go build -ldflags="-s -w" -trimpath -o bin/tokman ./cmd/tokman
	@echo "Binary size: $$(du -h bin/tokman | cut -f1)"

# Multi-platform build
build-all:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o bin/tokman-linux-amd64 ./cmd/tokman
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -trimpath -o bin/tokman-linux-arm64 ./cmd/tokman
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o bin/tokman-darwin-amd64 ./cmd/tokman
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -trimpath -o bin/tokman-darwin-arm64 ./cmd/tokman
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o bin/tokman-windows-amd64.exe ./cmd/tokman

test:
	go test -race -count=1 ./...

test-short:
	go test -short -count=1 ./...

test-cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

typecheck:
	go vet ./...

vet:
	go vet ./...

fmt:
	gofmt -s -w .
	goimports -w .

clean:
	rm -rf bin/ coverage.out coverage.html

benchmark:
	go test -bench=. -benchmem ./...

# Run all checks
check: fmt vet typecheck lint test

# Quick check (skip slow tests)
check-quick: fmt vet typecheck lint test-short

# CI check (what CI runs)
ci: test lint benchmark
