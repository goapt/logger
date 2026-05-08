package sloghttp

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

type customAttributesCtxKeyType struct{}

var customAttributesCtxKey = customAttributesCtxKeyType{}

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
	Level slog.Level

	WithUserAgent      bool
	WithRequestBody    bool
	WithRequestHeader  bool
	WithResponseBody   bool
	WithResponseHeader bool

	Filters []Filter
}

var DefaultConfig = Config{
	Level:              slog.LevelInfo,
	WithUserAgent:      true,
	WithRequestBody:    true,
	WithRequestHeader:  true,
	WithResponseBody:   true,
	WithResponseHeader: true,

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
	ip := extractClientIP(r)
	referer := r.Referer()

	var baseAttributes = []slog.Attr{
		slog.String("host", host),
		slog.String("path", r.URL.Path),
		slog.String("method", method),
		slog.String("ip", ip),
		slog.Float64("request_duration", latency.Seconds()),
		slog.Int("http_status", status),
	}

	requestAttributes := []slog.Attr{
		slog.Time("time", start.UTC()),
		slog.String("query", r.URL.RawQuery),
		slog.String("referer", referer),
	}

	responseAttributes := []slog.Attr{
		slog.Time("time", end.UTC()),
	}

	if err != nil {
		baseAttributes = append(baseAttributes, slog.Any("http_error", err))
	}

	reqID := GetRequestID(r)
	if reqID != "" {
		baseAttributes = append(baseAttributes, slog.String(RequestIDKey, reqID))
	}

	baseAttributes = append(baseAttributes, extractTraceSpanID(r.Context())...)

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

	level := config.Level
	logger.LogAttrs(r.Context(), level, strconv.Itoa(status)+": "+http.StatusText(status), attributes...)
}

func extractClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		firstIP := strings.TrimSpace(strings.Split(xff, ",")[0])
		if firstIP != "" {
			return firstIP
		}
	}

	if xRealIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); xRealIP != "" {
		return xRealIP
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}

	return r.RemoteAddr
}

func ensureRequestID(r *http.Request) *http.Request {
	requestID := r.Header.Get(RequestIDHeaderKey)
	if requestID == "" {
		requestID = uuid.NewString()
		r.Header.Set(RequestIDHeaderKey, requestID)
	}

	return r
}

// GetRequestID returns the request identifier.
func GetRequestID(r *http.Request) string {
	if r == nil {
		return ""
	}
	return r.Header.Get(RequestIDHeaderKey)
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

func extractTraceSpanID(ctx context.Context) []slog.Attr {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return []slog.Attr{}
	}

	var attrs []slog.Attr
	spanCtx := span.SpanContext()

	// 只要 span 处于 recording 状态，就记录 trace_id / span_id。
	if spanCtx.HasTraceID() {
		traceID := trace.SpanFromContext(ctx).SpanContext().TraceID().String()
		attrs = append(attrs, slog.String(TraceIDKey, traceID))
	}

	if spanCtx.HasSpanID() {
		spanID := spanCtx.SpanID().String()
		attrs = append(attrs, slog.String(SpanIDKey, spanID))
	}

	return attrs
}
