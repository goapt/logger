package sloghttp

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewMiddleware_RequestID_Logging_CustomAttr(t *testing.T) {
	h := &captureHandler{}
	logger := slog.New(h)

	cfg := DefaultConfig
	cfg.WithRequestID = true
	cfg.WithRequestBody = true
	cfg.WithResponseBody = true

	mw := NewMiddleware(logger, cfg)

	var capturedID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = GetRequestID(r)
		AddContextAttributes(r.Context(), slog.String("foo", "bar"))
		_, ok := r.Body.(*bodyReader)
		require.True(t, ok)
		_, _ = w.Write([]byte("ok"))
	})

	handler := mw(next)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://example.com/path", io.NopCloser(bytes.NewBufferString("req-body")))

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "ok", rec.Body.String())

	idHdr := req.Header.Get(RequestIDHeaderKey)
	require.NotEmpty(t, idHdr)
	require.Equal(t, idHdr, capturedID)

	require.NotEmpty(t, h.records)
	var top slogRecord
	for _, r := range h.records {
		if r.Msg != "" {
			top = r
			break
		}
	}
	require.NotEmpty(t, top.Attrs)

	_, hasReqGroup := findAttr(top.Attrs, "request")
	_, hasResGroup := findAttr(top.Attrs, "response")
	require.True(t, hasReqGroup)
	require.True(t, hasResGroup)

	ri, ok := findAttr(top.Attrs, RequestIDKey)
	require.True(t, ok)
	require.Equal(t, idHdr, ri.Value.String())

	foo, ok := findAttr(top.Attrs, "foo")
	require.True(t, ok)
	require.Equal(t, "bar", foo.Value.String())
}
