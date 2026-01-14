package sloghttp

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type customAttributesCtxKeyType struct{}
type requestIDCtxKeyType struct{}

var customAttributesCtxKey = customAttributesCtxKeyType{}
var requestIDCtxKey = requestIDCtxKeyType{}

var (
	TraceIDKey   = "trace_id"
	SpanIDKey    = "span_id"
	RequestIDKey = "request_id"

	RequestBodyMaxSize  = 64 * 1024 // 64KB
	ResponseBodyMaxSize = 64 * 1024 // 64KB

	HiddenRequestHeaders = map[string]struct{}{
		"authorization": {},
		"cookie":        {},
		"set-cookie":    {},
		"x-auth-token":  {},
		"x-csrf-token":  {},
		"x-xsrf-token":  {},
	}
	HiddenResponseHeaders = map[string]struct{}{
		"set-cookie": {},
	}

	// RequestIDHeaderKey Formatted with http.CanonicalHeaderKey
	RequestIDHeaderKey = "X-Request-Id"
)

type Config struct {
	DefaultLevel     slog.Level
	ClientErrorLevel slog.Level
	ServerErrorLevel slog.Level

	WithUserAgent      bool
	WithRequestID      bool
	WithRequestBody    bool
	WithRequestHeader  bool
	WithResponseBody   bool
	WithResponseHeader bool
	WithSpanID         bool
	WithTraceID        bool

	Filters []Filter
}

var DefaultConfig = Config{
	DefaultLevel:     slog.LevelInfo,
	ClientErrorLevel: slog.LevelWarn,
	ServerErrorLevel: slog.LevelError,

	WithUserAgent:      false,
	WithRequestID:      true,
	WithRequestBody:    false,
	WithRequestHeader:  false,
	WithResponseBody:   false,
	WithResponseHeader: false,
	WithSpanID:         false,
	WithTraceID:        false,

	Filters: []Filter{},
}

func log(logger *slog.Logger, config Config, r *http.Request, wr WrapResponse, br *bodyReader, start time.Time, err error) {
	for _, filter := range config.Filters {
		if !filter(wr, r) {
			return
		}
	}

	status := wr.Status()
	method := r.Method
	host := r.Host
	end := time.Now()
	latency := end.Sub(start)
	userAgent := r.UserAgent()
	ip := r.RemoteAddr
	referer := r.Referer()

	var baseAttributes []slog.Attr

	requestAttributes := []slog.Attr{
		slog.Time("time", start.UTC()),
		slog.String("method", method),
		slog.String("host", host),
		slog.String("path", r.URL.Path),
		slog.String("query", r.URL.RawQuery),
		slog.String("ip", ip),
		slog.String("referer", referer),
	}

	responseAttributes := []slog.Attr{
		slog.Time("time", end.UTC()),
		slog.Duration("latency", latency),
		slog.Int("status", status),
	}

	if err != nil {
		responseAttributes = append(responseAttributes, slog.Any("http_error", err))
	}

	if config.WithRequestID {
		reqID := GetRequestIDFromContext(r.Context())
		if reqID != "" {
			baseAttributes = append(baseAttributes, slog.String(RequestIDKey, reqID))
		}
	}

	baseAttributes = append(baseAttributes, extractTraceSpanID(r.Context(), config.WithTraceID, config.WithSpanID)...)

	if br != nil {
		requestAttributes = append(requestAttributes, slog.Int("length", br.bytes))
		if config.WithRequestBody {
			if br.body != nil {
				requestAttributes = append(requestAttributes, slog.String("body", br.body.String()))
			}
		}
	}

	if config.WithRequestHeader {
		var kv []any
		for k, v := range r.Header {
			if _, found := HiddenRequestHeaders[strings.ToLower(k)]; found {
				continue
			}
			kv = append(kv, slog.Any(k, v))
		}
		requestAttributes = append(requestAttributes, slog.Group("header", kv...))
	}

	if config.WithUserAgent {
		requestAttributes = append(requestAttributes, slog.String("user-agent", userAgent))
	}

	responseAttributes = append(responseAttributes, slog.Int("length", wr.BytesWritten()))
	if config.WithResponseBody {
		body := wr.Body()
		if body != nil {
			responseAttributes = append(responseAttributes, slog.String("body", string(body)))
		}
	}

	if config.WithResponseHeader {
		var kv []any
		for k, v := range wr.Header() {
			if _, found := HiddenResponseHeaders[strings.ToLower(k)]; found {
				continue
			}
			kv = append(kv, slog.Any(k, v))
		}
		responseAttributes = append(responseAttributes, slog.Group("header", kv...))
	}

	attributes := append(
		[]slog.Attr{
			{
				Key:   "request",
				Value: slog.GroupValue(requestAttributes...),
			},
			{
				Key:   "response",
				Value: slog.GroupValue(responseAttributes...),
			},
		},
		baseAttributes...,
	)

	if v := r.Context().Value(customAttributesCtxKey); v != nil {
		if m, ok := v.(*sync.Map); ok {
			m.Range(func(key, value any) bool {
				attributes = append(attributes, slog.Attr{Key: key.(string), Value: value.(slog.Value)})
				return true
			})
		}
	}

	level := config.DefaultLevel
	if status >= http.StatusInternalServerError {
		level = config.ServerErrorLevel
	} else if status >= http.StatusBadRequest {
		level = config.ClientErrorLevel
	}

	logger.LogAttrs(r.Context(), level, strconv.Itoa(status)+": "+http.StatusText(status), attributes...)
}

// GetRequestID returns the request identifier.
func GetRequestID(r *http.Request) string {
	return GetRequestIDFromContext(r.Context())
}

// GetRequestIDFromContext returns the request identifier from the context.
func GetRequestIDFromContext(ctx context.Context) string {
	requestID := ctx.Value(requestIDCtxKey)
	if id, ok := requestID.(string); ok {
		return id
	}

	return ""
}

// NewContextAttributes creates a new context with custom attributes.
func NewContextAttributes(ctx context.Context, attrs ...slog.Attr) context.Context {
	if v := ctx.Value(customAttributesCtxKey); v == nil {
		ctx = context.WithValue(ctx, customAttributesCtxKey, &sync.Map{})
		AddContextAttributes(ctx, attrs...)
	} else {
		AddContextAttributes(ctx, attrs...)
	}

	return ctx
}

// AddContextAttributes add custom attributes to the context, context must be created by NewContextAttributes.
func AddContextAttributes(ctx context.Context, attrs ...slog.Attr) {
	if v := ctx.Value(customAttributesCtxKey); v != nil {
		if m, ok := v.(*sync.Map); ok {
			for _, attr := range attrs {
				m.Store(attr.Key, attr.Value)
			}
		}
	}
}

func extractTraceSpanID(ctx context.Context, withTraceID bool, withSpanID bool) []slog.Attr {
	if !(withTraceID || withSpanID) {
		return []slog.Attr{}
	}

	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return []slog.Attr{}
	}

	var attrs []slog.Attr
	spanCtx := span.SpanContext()

	if withTraceID && spanCtx.HasTraceID() {
		traceID := trace.SpanFromContext(ctx).SpanContext().TraceID().String()
		attrs = append(attrs, slog.String(TraceIDKey, traceID))
	}

	if withSpanID && spanCtx.HasSpanID() {
		spanID := spanCtx.SpanID().String()
		attrs = append(attrs, slog.String(SpanIDKey, spanID))
	}

	return attrs
}
