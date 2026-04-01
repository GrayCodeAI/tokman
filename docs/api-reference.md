# API Reference

Complete REST API documentation for TokMan server.

## Smart Context Endpoints

TokMan exposes smart context delivery over the MCP server and dashboard APIs.

### MCP `POST /read`

Reads a file with TokMan's smart context modes.

Request body:

```json
{
  "path": "internal/server/server.go",
  "mode": "graph",
  "max_tokens": 400,
  "related_files": 4,
  "save_snapshot": true
}
```

Supported modes:
- `auto`
- `full`
- `map`
- `signatures`
- `aggressive`
- `entropy`
- `lines`
- `delta`
- `graph`

Response fields:
- `content`
- `original_tokens`
- `final_tokens`
- `saved_tokens`
- `reduction_percent`

### MCP `POST /bundle`

Returns a graph-aware context bundle for a target file.

Request body:

```json
{
  "path": "internal/server/server.go",
  "mode": "graph",
  "related_files": 4,
  "max_tokens": 500
}
```

Response fields:
- `path`
- `related_files`
- `content`
- `original_tokens`
- `final_tokens`
- `saved_tokens`
- `reduction_percent`

Use `/bundle` when an agent needs target-file context plus project-selected neighbors in one request.

### Dashboard `GET /api/context-reads`

Returns recent smart-read activity from tracking.

Optional query parameter:
- `kind=read|delta|mcp`
- `mode=auto|full|map|signatures|aggressive|entropy|lines|delta|graph`

### Dashboard `GET /api/context-read-summary`

Returns aggregate smart-read savings grouped by:
- `read`
- `delta`
- `mcp`

### Dashboard `GET /api/context-read-trend`

Returns daily smart-read savings over time.

### Dashboard `GET /api/context-read-top-files`

Returns the highest-value files by smart-read savings.

### Dashboard `GET /api/context-read-projects`

Returns the highest-value projects by smart-read savings.

### Dashboard `GET /api/context-read-quality`

Returns mode-level smart-read quality metrics, including:
- `mode`
- `commands`
- `tokens_saved`
- `reduction_pct`
- `avg_delivered_tokens`
- `avg_saved_tokens`
- `avg_related_files`

This endpoint is backed by structured tracking fields (`context_kind`, `context_mode`, `context_resolved_mode`, `context_target`, `context_bundle`) rather than command-name parsing.

## Base URL

```
http://localhost:8080
```

## Authentication

Currently, TokMan does not require authentication. For production deployments, consider using a reverse proxy with authentication.

## Endpoints

### Health Check

**GET** `/health`

Returns server health status.

**Response**
```json
{
  "status": "ok",
  "version": "1.2.0"
}
```

**Example**
```bash
curl http://localhost:8080/health
```

---

### Compress Text

**POST** `/compress`

Compress text using the specified mode.

**Request Body**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `input` | string | Yes | Text to compress |
| `mode` | string | No | Compression mode: `conservative`, `balanced`, `aggressive` (default: `balanced`) |
| `budget` | int | No | Target token count (default: 4000) |

**Response**
```json
{
  "output": "compressed text",
  "original_tokens": 100,
  "final_tokens": 50,
  "tokens_saved": 50,
  "reduction_percent": 50.0,
  "processing_time_ms": 5
}
```

**Example**
```bash
curl -X POST http://localhost:8080/compress \
  -H "Content-Type: application/json" \
  -d '{
    "input": "Your long text here...",
    "mode": "aggressive"
  }'
```

---

### Adaptive Compression

**POST** `/compress/adaptive`

Compress text with automatic content type detection and optimization.

**Request Body**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `input` | string | Yes | Text to compress |
| `mode` | string | No | Compression mode (default: `balanced`) |

**Response**
```json
{
  "output": "compressed text",
  "original_tokens": 100,
  "final_tokens": 45,
  "tokens_saved": 55,
  "reduction_percent": 55.0,
  "processing_time_ms": 8
}
```

**Example**
```bash
curl -X POST http://localhost:8080/compress/adaptive \
  -H "Content-Type: application/json" \
  -d '{"input": "function main() { return 42; }"}'
```

---

### Analyze Content

**POST** `/analyze`

Analyze content type without compression.

**Request Body**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `input` | string | Yes | Text to analyze |

**Response**
```json
{
  "content_type": "code"
}
```

**Content Types**
- `code` - Source code, scripts
- `conversation` - Chat logs, dialogue
- `logs` - System logs, error messages
- `documents` - Articles, documentation
- `mixed` - Mixed content

**Example**
```bash
curl -X POST http://localhost:8080/analyze \
  -H "Content-Type: application/json" \
  -d '{"input": "function main() { return 42; }"}'
```

---

### Server Statistics

**GET** `/stats`

Returns server statistics and metrics.

**Response**
```json
{
  "version": "1.2.0",
  "layer_count": 14,
  "total_requests": 1000,
  "total_compressions": 950,
  "total_tokens_saved": 45000,
  "avg_reduction_pct": 35.5
}
```

**Example**
```bash
curl http://localhost:8080/stats
```

---

### Prometheus Metrics

**GET** `/metrics`

Returns metrics in Prometheus text format for scraping.

**Response**
```
# HELP tokman_requests_total Total number of requests
# TYPE tokman_requests_total counter
tokman_requests_total 1000

# HELP tokman_compressions_total Total number of compressions
# TYPE tokman_compressions_total counter
tokman_compressions_total 950

# HELP tokman_tokens_saved_total Total tokens saved
# TYPE tokman_tokens_saved_total counter
tokman_tokens_saved_total 45000

# HELP tokman_avg_reduction_pct Average reduction percentage
# TYPE tokman_avg_reduction_pct gauge
tokman_avg_reduction_pct 35.5

# HELP tokman_uptime_seconds Server uptime in seconds
# TYPE tokman_uptime_seconds gauge
tokman_uptime_seconds 3600
```

**Example**
```bash
curl http://localhost:8080/metrics
```

---

## Error Responses

All errors follow this format:

```json
{
  "error": "Error message",
  "code": "ERROR_CODE"
}
```

**HTTP Status Codes**

| Code | Description |
|------|-------------|
| 200 | Success |
| 400 | Bad Request - Invalid input |
| 405 | Method Not Allowed |
| 500 | Internal Server Error |

---

## Rate Limiting

Currently, TokMan does not implement rate limiting. For production, consider using a reverse proxy.

---

## Content Types

| Type | Description | Recommended Mode |
|------|-------------|------------------|
| `code` | Source code, config files | `conservative` |
| `conversation` | Chat logs, Q&A | `balanced` |
| `logs` | System logs, errors | `aggressive` |
| `documents` | Articles, docs | `balanced` |
| `mixed` | Mixed content | `balanced` |
