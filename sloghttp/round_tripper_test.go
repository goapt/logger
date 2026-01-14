package sloghttp

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type dummyRT struct{}

func (d dummyRT) RoundTrip(r *http.Request) (*http.Response, error) {
	res := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewBufferString("resp-body")),
		Request:    r,
	}
	return res, nil
}

func TestNewRoundTripper_RequestIDAndLogging(t *testing.T) {
	h := &captureHandler{}
	logger := slog.New(h)

	cfg := DefaultConfig
	cfg.WithRequestID = true
	cfg.WithRequestBody = true
	cfg.WithResponseBody = true

	rt := NewRoundTripper(logger, dummyRT{}, cfg)

	req, err := http.NewRequest(http.MethodPost, "http://example.com/api", io.NopCloser(bytes.NewBufferString("req-body")))
	require.NoError(t, err)
	req = req.WithContext(context.Background())

	res, err := rt.RoundTrip(req)
	require.NoError(t, err)
	require.NotNil(t, res)

	idHdr := req.Header.Get(RequestIDHeaderKey)
	require.NotEmpty(t, idHdr)
	require.Equal(t, idHdr, GetRequestID(res.Request))

	_, ok := res.Request.Body.(*bodyReader)
	require.True(t, ok)

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
}

type errorRT struct{}

func (d errorRT) RoundTrip(r *http.Request) (*http.Response, error) {
	return nil, io.ErrUnexpectedEOF
}

func TestRoundTripper_Error(t *testing.T) {
	h := &captureHandler{}
	logger := slog.New(h)

	cfg := DefaultConfig
	cfg.WithResponseBody = true

	rt := NewRoundTripper(logger, errorRT{}, cfg)

	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)

	res, err := rt.RoundTrip(req)
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	require.Nil(t, res)

	require.NotEmpty(t, h.records)
	var top slogRecord
	for _, r := range h.records {
		if r.Msg != "" {
			top = r
			break
		}
	}
	require.NotEmpty(t, top.Attrs)

	resGroup, ok := findAttr(top.Attrs, "response")
	require.True(t, ok)

	// t.Logf("Response Group Attrs: %+v", resGroup.Value.Group())

	errAttr, ok := findAttr(resGroup.Value.Group(), "http_error")
	require.True(t, ok)
	require.Equal(t, io.ErrUnexpectedEOF, errAttr.Value.Any())
}
