# TokMan Production Deployment Guide

## Pre-Deployment Checklist

### 1. Configuration

```toml
# /etc/tokman/config.toml

[tracking]
enabled = true
database_path = "/var/lib/tokman/tokman.db"

[filter]
mode = "balanced"

[pipeline]
# Production limits
max_context_tokens = 2000000
stream_threshold = 500000

# Resilience
failsafe_mode = true
validate_output = true
tee_on_failure = true

# Caching for performance
cache_enabled = true
cache_max_size = 10000

# Budget enforcement
default_budget = 0  # Unlimited by default
hard_budget_limit = true

[hooks]
audit_dir = "/var/log/tokman/audit"
excluded_commands = ["tokman", "sudo"]

[dashboard]
port = 8080
bind = "127.0.0.1"  # Localhost only

[alerts]
enabled = true
daily_token_limit = 10000000  # 10M tokens/day
weekly_token_limit = 50000000  # 50M tokens/week
usage_spike_threshold = 3.0
```

### 2. System Requirements

| Resource | Minimum | Recommended |
|----------|---------|-------------|
| CPU | 1 core | 2+ cores |
| RAM | 256MB | 1GB |
| Disk | 100MB | 1GB (for tracking DB) |
| Go | 1.26+ | 1.26+ (SIMD support) |

### 3. Security Considerations

- **Run as non-root user**: Create `tokman` user
- **File permissions**: Config 640, DB 600
- **Network**: Bind to localhost, use reverse proxy for external access
- **API Key**: Set `TOKMAN_API_KEY` for server mode

```bash
# Create tokman user
sudo useradd -r -s /bin/false tokman

# Set up directories
sudo mkdir -p /var/lib/tokman /var/log/tokman /etc/tokman
sudo chown tokman:tokman /var/lib/tokman /var/log/tokman

# Set permissions
sudo chmod 750 /var/lib/tokman /var/log/tokman
```

## Deployment Methods

### Docker

```dockerfile
# Dockerfile
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -ldflags="-s -w" -o tokman ./cmd/tokman

FROM alpine:3.19
RUN adduser -D -g '' tokman
COPY --from=builder /app/tokman /usr/local/bin/tokman
USER tokman
EXPOSE 8080
ENTRYPOINT ["tokman"]
```

```yaml
# docker-compose.yml
version: '3.8'
services:
  tokman:
    build: .
    ports:
      - "127.0.0.1:8080:8080"
    volumes:
      - tokman-data:/var/lib/tokman
      - tokman-logs:/var/log/tokman
      - ./config.toml:/etc/tokman/config.toml:ro
    environment:
      - TOKMAN_API_KEY=${TOKMAN_API_KEY}
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "tokman", "doctor"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  tokman-data:
  tokman-logs:
```

### Kubernetes

```yaml
# k8s-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tokman
spec:
  replicas: 2
  selector:
    matchLabels:
      app: tokman
  template:
    metadata:
      labels:
        app: tokman
    spec:
      containers:
      - name: tokman
        image: tokman:latest
        ports:
        - containerPort: 8080
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        volumeMounts:
        - name: config
          mountPath: /etc/tokman
          readOnly: true
        - name: data
          mountPath: /var/lib/tokman
      volumes:
      - name: config
        configMap:
          name: tokman-config
      - name: data
        emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: tokman
spec:
  selector:
    app: tokman
  ports:
  - port: 8080
    targetPort: 8080
```

### Systemd Service

```ini
# /etc/systemd/system/tokman.service
[Unit]
Description=TokMan Token Compression Service
After=network.target

[Service]
Type=simple
User=tokman
Group=tokman
ExecStart=/usr/local/bin/tokman serve --config /etc/tokman/config.toml
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

# Security
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/tokman /var/log/tokman

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable tokman
sudo systemctl start tokman
```

## Monitoring

### Health Endpoints

| Endpoint | Purpose |
|----------|---------|
| `/health` | Basic liveness check |
| `/health/ready` | Readiness (DB connected, config loaded) |
| `/metrics` | Prometheus metrics |

### Prometheus Metrics

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'tokman'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: /metrics
```

### Key Metrics to Monitor

- `tokman_commands_total` - Commands processed
- `tokman_tokens_saved_total` - Total tokens saved
- `tokman_compression_ratio` - Average compression
- `tokman_errors_total` - Error count
- `tokman_latency_seconds` - Processing latency

### Alerts (Alertmanager)

```yaml
groups:
- name: tokman
  rules:
  - alert: TokManHighErrorRate
    expr: rate(tokman_errors_total[5m]) > 0.1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "TokMan error rate high"

  - alert: TokManTokenLimitExceeded
    expr: tokman_daily_tokens > 10000000
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "Daily token limit exceeded"
```

## Performance Tuning

### Large Context (2M+ tokens)

```toml
[pipeline]
stream_threshold = 500000  # Stream processing
chunk_size = 100000        # 100K chunks
parallel_layers = false    # Sequential for stability
cache_enabled = true
cache_max_size = 10000
```

### High Throughput

```toml
[pipeline]
preset = "fast"           # Only 3 layers
parallel_layers = true    # Parallel processing
cache_enabled = true
cache_max_size = 50000
```

### Memory Constrained

```toml
[pipeline]
stream_threshold = 100000  # Earlier streaming
chunk_size = 50000         # Smaller chunks
cache_max_size = 1000
enable_sketch_store = true # 90% memory reduction
```

## Troubleshooting

### Common Issues

**Database locked**
```bash
# Check for multiple instances
ps aux | grep tokman

# Check database integrity
sqlite3 /var/lib/tokman/tokman.db "PRAGMA integrity_check;"
```

**High memory usage**
```bash
# Enable streaming
tokman --stream-threshold=100000

# Use sketch store
tokman --enable-sketch-store
```

**Slow compression**
```bash
# Use fast preset
tokman --preset=fast

# Disable expensive layers
tokman --disable-perplexity --disable-hierarchical
```

### Debug Mode

```bash
# Enable debug logging
TOKMAN_LOG_LEVEL=debug tokman serve

# Check logs
journalctl -u tokman -f
```

## Backup & Recovery

### Backup

```bash
# Backup database
cp /var/lib/tokman/tokman.db /backup/tokman-$(date +%Y%m%d).db

# Backup config
cp /etc/tokman/config.toml /backup/
```

### Recovery

```bash
# Restore database
systemctl stop tokman
cp /backup/tokman-20260324.db /var/lib/tokman/tokman.db
systemctl start tokman
```

## Security Checklist

- [ ] Run as non-root user
- [ ] Config file permissions 640
- [ ] Database permissions 600
- [ ] API key set for server mode
- [ ] Dashboard bound to localhost
- [ ] Rate limiting configured
- [ ] TLS enabled via reverse proxy
- [ ] Audit logging enabled
- [ ] No secrets in config files
