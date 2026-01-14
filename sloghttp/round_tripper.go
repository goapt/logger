package sloghttp

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type RoundTripper struct {
	next   http.RoundTripper
	logger *slog.Logger
	config Config
}

func (rt *RoundTripper) RoundTrip(r *http.Request) (res *http.Response, err error) {
	start := time.Now()

	requestID := r.Header.Get(RequestIDHeaderKey)
	if rt.config.WithRequestID {
		if requestID == "" {
			requestID = uuid.New().String()
			r.Header.Set(RequestIDHeaderKey, requestID)
		}
		r = r.WithContext(context.WithValue(r.Context(), requestIDCtxKey, requestID))
	}

	// dump request body
	var br *bodyReader
	if r.Body != nil {
		br = newBodyReader(r.Body, RequestBodyMaxSize, rt.config.WithRequestBody)
		r.Body = br
	}

	// Make sure we create a map only once per request (in case we have multiple middleware instances)
	if v := r.Context().Value(customAttributesCtxKey); v == nil {
		r = r.WithContext(NewContextAttributes(r.Context()))
	}

	// dump response body
	// bw := newBodyWriter(w, ResponseBodyMaxSize, config.WithResponseBody)
	res, err = rt.next.RoundTrip(r)

	var bw WrapResponse
	if err != nil {
		// 如果请求失败，使用空响应包装器，确保 log 函数能正常执行
		bw = noopResponse{}
	} else {
		bw = newResponse(res, RequestBodyMaxSize, rt.config.WithResponseBody)
	}

	defer func() {
		log(rt.logger, rt.config, r, bw, br, start, err)
	}()

	return res, err
}

// NewRoundTripper returns a `http.RoundTripper` that logs requests using slog.
func NewRoundTripper(logger *slog.Logger, next http.RoundTripper, config Config) *RoundTripper {
	return &RoundTripper{
		next:   next,
		logger: logger,
		config: config,
	}
}
