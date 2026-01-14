# Logger

<p>
<a href="https://github.com/goapt/logger/actions"><img src="https://github.com/goapt/logger/workflows/build/badge.svg" alt="Build Status"></a>
<a href="https://codecov.io/gh/goapt/logger"><img src="https://codecov.io/gh/goapt/logger/branch/master/graph/badge.svg" alt="codecov"></a>
<a href="https://goreportcard.com/report/github.com/goapt/logger"><img src="https://goreportcard.com/badge/github.com/goapt/logger" alt="Go Report Card
"></a>
<a href="https://godoc.org/github.com/goapt/logger"><img src="https://godoc.org/github.com/goapt/logger?status.svg" alt="GoDoc"></a>
<a href="https://opensource.org/licenses/mit-license.php" rel="nofollow"><img src="https://badges.frapsoft.com/os/mit/mit.svg?v=103"></a>
</p>

Golang logger,use log/slog, support logroate

## Default logger

default log out to os.Stderr

```go
import "github.com/goapt/logger"

func main(){
    logger.New(&logger.Config{
        Mode: logger.ModeStd,
        Level: slog.LevelInfo,
    })
}
```

## Setting default logger

```
logger.SetDefault(logger.New(&logger.Config{
    Mode: logger.ModeFile,
    Level: slog.LevelInfo,
    FileName: "app.log",
    Detail: true,
}))
```

## Usage logger

```
// global logger
logger.Default().Info("this is new log",slog.String("key","value"))

// or
func Foo(log *slog.Logger) {
    log.Info("this is new log",slog.String("key","value"))
}
Foo(logger.Default())
```

## Print filename and line no

if `Detail` is true,the log data add filename and line no

```
{"file":"/Users/fifsky/wwwroot/go/library/src/github.com/fifsky/goblog/handler/index.go","func":"handler.IndexGet","level":"debug","line":16,"msg":"[test]","time":"2018-08-02 22:37:02"}
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
    logger.SetDefault(logger.New(&logger.Config{
        Mode:   logger.ModeStd,
        Level:  slog.LevelInfo,
        Detail: false,
    }))

    // Configure HTTP logging middleware (enable fields as needed)
    cfg := sloghttp.DefaultConfig
    cfg.WithRequestID = true       // auto generate/propagate X-Request-Id
    cfg.WithRequestBody = false    // capture request body (off by default; truncated)
    cfg.WithResponseBody = false   // capture response body (off by default; truncated)
    cfg.WithRequestHeader = false  // include request headers (sensitive headers redacted)
    cfg.WithResponseHeader = false // include response headers (sensitive headers redacted)
    cfg.WithTraceID = false        // extract TraceID from Context (requires OTel)
    cfg.WithSpanID = false         // extract SpanID from Context (requires OTel)

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
    "log/slog"
    "net/http"
    "strings"

    "github.com/goapt/logger"
    "github.com/goapt/logger/sloghttp"
)

func main() {
    logger.SetDefault(logger.New(&logger.Config{
        Mode:  logger.ModeStd,
        Level: slog.LevelInfo,
    }))

    cfg := sloghttp.DefaultConfig
    cfg.WithRequestID = true     // propagate/generate X-Request-Id for correlation
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

- OpenTelemetry integration: if Trace/Span are in Context, enable WithTraceID/WithSpanID to include them in logs.

```go
cfg := sloghttp.DefaultConfig
cfg.WithTraceID = true
cfg.WithSpanID = true
```

## Config Fields

- DefaultLevel / ClientErrorLevel / ServerErrorLevel: control log level mapping
- WithUserAgent / WithRequestID / WithRequestBody / WithRequestHeader: request switches
- WithResponseBody / WithResponseHeader: response switches
- WithTraceID / WithSpanID: extract Trace/Span from Context
- Filters: plug Accept*/Ignore* filters (method, path, status, host, etc.)

Filter example:

```go
cfg.Filters = []sloghttp.Filter{
    sloghttp.IgnorePathPrefix("/health", "/metrics"),
    sloghttp.IgnoreMethod(http.MethodOptions),
    sloghttp.AcceptStatusGreaterThanOrEqual(http.StatusBadRequest),
}
```