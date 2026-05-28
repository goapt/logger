# Logger

<p>
<a href="https://github.com/goapt/logger/actions"><img src="https://github.com/goapt/logger/workflows/build/badge.svg" alt="Build Status"></a>
<a href="https://codecov.io/gh/goapt/logger"><img src="https://codecov.io/gh/goapt/logger/branch/master/graph/badge.svg" alt="codecov"></a>
<a href="https://goreportcard.com/report/github.com/goapt/logger"><img src="https://goreportcard.com/badge/github.com/goapt/logger" alt="Go Report Card
"></a>
<a href="https://godoc.org/github.com/goapt/logger"><img src="https://godoc.org/github.com/goapt/logger?status.svg" alt="GoDoc"></a>
<a href="https://opensource.org/licenses/mit-license.php" rel="nofollow"><img src="https://badges.frapsoft.com/os/mit/mit.svg?v=103"></a>
</p>

Golang logger, use `log/slog`, support multi-handler and log rotate.

## Quick Start

```go
import (
    "log/slog"
    "os"

    "github.com/goapt/logger"
)

func main() {
    // Default logger: JSON to stdout at Info level
    logger.Info("hello", slog.String("key", "value"))
}
```

## Create Logger

`New` accepts one or more `slog.Handler` and broadcasts log records to all of them.

```go
// Simple — stdout only (default level: Info)
l := logger.New()

// Single handler with custom level
l := logger.New(logger.NewJSONHandler(os.Stdout, logger.WithLevel(slog.LevelDebug)))

// Multiple handlers with independent levels
l := logger.New(
    logger.NewJSONHandler(os.Stdout, logger.WithLevel(slog.LevelDebug)),          // debug → stdout
    logger.NewJSONHandler(logger.NewFileWriter("app.log"), logger.WithLevel(slog.LevelInfo)), // info+ → file
)

l.Debug("debug msg") // stdout only
l.Info("info msg")   // stdout + file
l.Error("error msg") // stdout + file
```

## NewJSONHandler

Creates a JSON handler writing to any `io.Writer`.

```go
logger.NewJSONHandler(os.Stdout)                                              // default: Info level
logger.NewJSONHandler(os.Stdout, logger.WithLevel(slog.LevelDebug))           // custom level
logger.NewJSONHandler(os.Stdout, logger.WithSource())                         // add file:line
logger.NewJSONHandler(os.Stdout, logger.WithLevel(slog.LevelDebug), logger.WithSource())
```

### HandlerOption

| Option                | Description                                   |
| --------------------- | --------------------------------------------- |
| `WithLevel(level)`    | Minimum log level (default: `slog.LevelInfo`) |
| `WithSource()`        | Include source file path and line number      |
| `WithReplaceAttr(fn)` | Custom attribute replacement function         |

## NewFileWriter

Creates a rotating file writer (implements `io.WriteCloser`).

```go
w := logger.NewFileWriter("app.log")
defer w.Close()

// With options
w := logger.NewFileWriter("app.log",
    logger.WithMaxSize(100 * 1024 * 1024), // 100 MB per file (default: 200 MB)
    logger.WithMaxFiles(5),                 // keep 5 backups (default: 3)
    logger.WithMaxAge(7),                   // retain 7 days (default: 3)
)
```

### FileOption

| Option               | Description                                     |
| -------------------- | ----------------------------------------------- |
| `WithMaxSize(bytes)` | Max file size before rotation (default: 200 MB) |
| `WithMaxFiles(n)`    | Max number of backup files (default: 3)         |
| `WithMaxAge(days)`   | Max days to retain backups (default: 3)         |

## Setting Default Logger

```go
logger.SetDefault(logger.New(
    logger.NewJSONHandler(os.Stdout, logger.WithLevel(slog.LevelDebug)),
    logger.NewJSONHandler(logger.NewFileWriter("app.log"), logger.WithLevel(slog.LevelInfo), logger.WithSource()),
))
```

## Usage

```go
// Package-level convenience functions (use Default logger)
logger.Info("this is a log", slog.String("key", "value"))
logger.Debug("debug message")
logger.Error("something went wrong", slog.String("error", err.Error()))

// Pass logger to functions
func Foo(log *slog.Logger) {
    log.Info("this is new log", slog.String("key", "value"))
}
Foo(logger.Default())
```

## Output Format

JSON output with formatted timestamps:

```json
{
  "time": "2025-05-28 10:30:00.123",
  "level": "INFO",
  "msg": "hello",
  "key": "value"
}
```

With `WithSource()`:

```json
{
  "time": "2025-05-28 10:30:00.123",
  "level": "DEBUG",
  "msg": "[test]",
  "source": { "function": "main.foo", "file": "/app/main.go", "line": 42 }
}
```

# Record the logs of the HTTP Handler and HTTP Client

Use sloghttp to record HTTP server and client request/response data as structured logs for observability and search. Supports request ID correlation, request/response body capture with truncation, sensitive header redaction, flexible filters, and OpenTelemetry Trace/Span extraction. Code forked from https://github.com/samber/slog-http, with additional support for HTTP client logging.

## Server Handler Logging

```go
package main

import (
    "log/slog"
    "net/http"

    "github.com/goapt/logger"
    "github.com/goapt/logger/sloghttp"
)

func main() {
    // Initialize default logger (JSON to stdout)
    logger.SetDefault(logger.New(logger.NewJSONHandler(os.Stdout)))

    // Configure HTTP logging middleware (enable fields as needed)
    cfg := sloghttp.DefaultConfig
    cfg.Level = slog.LevelInfo     // log level for HTTP logs
    cfg.WithRequestBody = false    // capture request body (on by default; truncated)
    cfg.WithResponseBody = false   // capture response body (on by default; truncated)
    cfg.WithRequestHeader = false  // include request headers (sensitive headers redacted)
    cfg.WithResponseHeader = false // include response headers (sensitive headers redacted)

    // Optional: filter routes/methods/status codes you don’t want to log
    // cfg.Filters = []sloghttp.Filter{
    //     sloghttp.IgnorePathPrefix("/health", "/metrics"),
    //     sloghttp.IgnoreMethod(http.MethodOptions),
    // }

    mw := sloghttp.NewMiddleware(logger.Default(), cfg)

    mux := http.NewServeMux()
    mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
        // Enrich with business attributes via Context; included in logs
        ctx := sloghttp.NewContextAttributes(r.Context(), slog.String("user_id", "123"))
        r = r.WithContext(ctx)

        // Retrieve request ID for business log correlation
        reqID := sloghttp.GetRequestID(r)
        logger.Default().Info("handle hello", slog.String("request_id", reqID))

        w.Write([]byte("ok"))
    })

    // Start server with middleware
    _ = http.ListenAndServe(":8080", mw(mux))
}
```

## HTTP Client Logging

```go
package main

import (
    "context"
    "log/slog"
    "net/http"
    "strings"

    "github.com/goapt/logger"
    "github.com/goapt/logger/sloghttp"
)

func main() {
    logger.SetDefault(logger.New(logger.NewJSONHandler(os.Stdout)))

    cfg := sloghttp.DefaultConfig
    cfg.Level = slog.LevelInfo   // log level for HTTP logs
    cfg.WithRequestBody = true   // capture client request body
    cfg.WithResponseBody = true  // capture client response body

    // Wrap Transport with sloghttp RoundTripper
    rt := sloghttp.NewRoundTripper(logger.Default(), http.DefaultTransport, cfg)
    client := &http.Client{Transport: rt}

    ctx := sloghttp.NewContextAttributes(context.Background(), slog.String("order_id", "A1001"))
    req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "http://example.com/api", strings.NewReader("payload"))
    res, err := client.Do(req)
    if err != nil {
        logger.Default().Error("http request failed", slog.String("error", err.Error()))
        return
    }
    defer res.Body.Close()
}
```

## Advanced

- Sensitive headers are redacted by default: authorization/cookie/x-auth-token, etc.
- Customize capture size: adjust global max bytes for request/response body.

```go
// Max bytes captured for request/response bodies (default 64KB)
sloghttp.RequestBodyMaxSize = 128 * 1024
sloghttp.ResponseBodyMaxSize = 128 * 1024
```

- OpenTelemetry integration: when `span.IsRecording()` is true, `trace_id` and `span_id` are recorded automatically.

## Config Fields

- Level: control log level
- WithUserAgent / WithRequestBody / WithRequestHeader: request switches
- WithResponseBody / WithResponseHeader: response switches
- request_id: always generated/propagated via `X-Request-Id` header
- trace_id/span_id: auto extracted when `span.IsRecording()` is true
- Filters: plug Accept*/Ignore* filters (method, path, status, host, etc.)

Filter example:

```go
cfg.Filters = []sloghttp.Filter{
    sloghttp.IgnorePathPrefix("/health", "/metrics"),
    sloghttp.IgnoreMethod(http.MethodOptions),
    sloghttp.AcceptStatusGreaterThanOrEqual(http.StatusBadRequest),
}
```
