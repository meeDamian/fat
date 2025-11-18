# Beautiful Terminal Logging

The application now features beautiful, colored terminal output when running interactively.

## What You See Now

When you run `go run ./cmd/fat`, you'll see clean, colored output like this:

```
15:04 INF loading API keys
15:04 INF api keys loaded
15:04 INF initializing database
15:04 INF database initialized
15:04 INF starting background archiver interval=1h0m0s
15:04 INF starting server addr=:4444
15:04 INF http request method=GET path=/models status=200 duration=683µs ip=::1
15:04 INF http request method=GET path=/ws status=200 duration=4.02s ip=::1
15:04 INF model request model=grok-4-fast round=1
15:04 INF model response model=grok-4-fast tokens_in=1234 tokens_out=567 duration=2.3s
15:04 WRN api key missing family=mistral
15:04 WRN http request method=GET path=/notfound status=404 duration=74µs ip=::1
15:05 ERR http request method=POST path=/api/error status=500 duration=120ms ip=::1
```

### Colors

- 🔵 **INFO** - Blue (2xx status codes)
- 🟡 **WARN** - Yellow (4xx status codes)
- 🔴 **ERROR** - Red (5xx status codes)
- ⚪ **DEBUG** - Gray

### HTTP Request Logging

All HTTP requests are logged with:
- **method**: HTTP method (GET, POST, etc.)
- **path**: Request path
- **status**: HTTP status code
- **duration**: Request processing time
- **ip**: Client IP address

Log level is automatically chosen based on status code:
- **2xx/3xx** → INFO (blue)
- **4xx** → WARN (yellow)
- **5xx** → ERROR (red)

## Previous Output (Mixed Formats)

Before this change, you would see inconsistent formatting:

```json
{"time":"2025-11-18T15:04:32.123Z","level":"INFO","msg":"loading API keys"}
{"time":"2025-11-18T15:04:32.456Z","level":"INFO","msg":"api keys loaded"}
{"time":"2025-11-18T15:04:32.789Z","level":"INFO","msg":"initializing database"}
{"time":"2025-11-18T15:04:33.012Z","level":"INFO","msg":"database initialized"}
{"time":"2025-11-18T15:04:33.234Z","level":"INFO","msg":"starting background archiver","interval":"1h0m0s"}
{"time":"2025-11-18T15:04:33.456Z","level":"INFO","msg":"server starting","address":":4444"}
[GIN] 2025/11/18 - 15:04:35 | 200 |  4.022797612s |             ::1 | GET      "/ws"
[GIN] 2025/11/18 - 15:04:35 | 304 |     588.433µs |             ::1 | GET      "/"
[GIN] 2025/11/18 - 15:04:36 | 200 |      74.289µs |             ::1 | GET      "/question/random"
```

Notice the inconsistency: slog JSON + default Gin format! 😱

## Smart Auto-Detection

The logger automatically detects your environment:

### Terminal (TTY)
When running in a terminal (`go run ./cmd/fat`):
- ✅ Beautiful colored output
- ✅ Readable timestamps (24-hour format: `15:04`)
- ✅ Compact attribute display
- ✅ Source file locations (debug level only)

### Piped/Redirected Output
When output is piped or redirected (`go run ./cmd/fat | tee log.txt`):
- ✅ JSON format for easy parsing
- ✅ RFC3339 timestamps for precision
- ✅ Structured data for log aggregators
- ✅ Machine-readable format

## Configuration

Control log level with environment variable:

```bash
# Info level (default) - balanced output
go run ./cmd/fat

# Debug level - includes source file locations
FAT_LOG_LEVEL=debug go run ./cmd/fat

# Warning level - only warnings and errors
FAT_LOG_LEVEL=warn go run ./cmd/fat

# Error level - only errors
FAT_LOG_LEVEL=error go run ./cmd/fat
```

## Debug Mode Example

With `FAT_LOG_LEVEL=debug`, you get additional source file information:

```
15:04 DBG database query internal/db/db.go:123 query="SELECT * FROM conversations" duration=5ms
15:04 DBG websocket message sent internal/server/websocket.go:89 type=answer model=grok
15:04 INF model request model=grok-4-fast round=1
```

## Implementation

Uses the excellent [tint](https://github.com/lmittmann/tint) library for slog, which provides:
- Zero dependencies beyond stdlib slog
- Automatic TTY detection
- Beautiful ANSI colors
- High performance
- Full slog compatibility

## Production Deployment

When deploying to production (systemd, Docker, etc.), the logger automatically switches to JSON format for compatibility with log aggregators like:
- CloudWatch
- Datadog
- Elasticsearch/Kibana
- Grafana Loki
- Splunk

No configuration changes needed - it just works! 🎉
