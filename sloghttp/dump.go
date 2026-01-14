package sloghttp

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
)

var _ WrapResponse = (*bodyWriter)(nil)

type WrapResponse interface {
	Header() http.Header
	Status() int
	BytesWritten() int
	Body() []byte
}

var _ http.ResponseWriter = (*bodyWriter)(nil)
var _ http.Flusher = (*bodyWriter)(nil)
var _ http.Hijacker = (*bodyWriter)(nil)

type bodyWriter struct {
	http.ResponseWriter
	body    *bytes.Buffer
	maxSize int
	bytes   int
	status  int
}

// implements http.ResponseWriter
func (w *bodyWriter) Write(b []byte) (int, error) {
	length := len(b)

	if w.body != nil {
		if w.body.Len()+length > w.maxSize {
			w.body.Truncate(min(w.maxSize, length, w.body.Len()))
			w.body.Write(b[:min(w.maxSize-w.body.Len(), length)])
		} else {
			w.body.Write(b)
		}
	}
	w.bytes += length //nolint:staticcheck
	return w.ResponseWriter.Write(b)
}

// WriteHeader implements http.ResponseWriter
func (w *bodyWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Flush implements http.Flusher
func (w *bodyWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack implements http.Hijacker
func (w *bodyWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hi, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hi.Hijack()
	}

	return nil, nil, errors.New("Hijack not supported")
}

// Unwrap implements the ability to use underlying http.ResponseController
func (w *bodyWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *bodyWriter) Status() int {
	return w.status
}

func (w *bodyWriter) BytesWritten() int {
	return w.bytes
}

func (w *bodyWriter) Body() []byte {
	return w.body.Bytes()
}

func newBodyWriter(writer http.ResponseWriter, maxSize int, recordBody bool) *bodyWriter {
	var body *bytes.Buffer
	if recordBody {
		body = bytes.NewBufferString("")
	}

	return &bodyWriter{
		ResponseWriter: writer,
		body:           body,
		maxSize:        maxSize,
		bytes:          0,
		status:         http.StatusOK,
	}
}

type bodyReader struct {
	io.ReadCloser
	body    *bytes.Buffer
	maxSize int
	bytes   int
}

// implements io.Reader
func (r *bodyReader) Read(b []byte) (int, error) {
	n, err := r.ReadCloser.Read(b)
	if r.body != nil && r.body.Len() < r.maxSize {
		if r.body.Len()+n > r.maxSize {
			r.body.Write(b[:min(r.maxSize-r.body.Len(), n)])
		} else {
			r.body.Write(b[:n])
		}
	}
	r.bytes += n
	return n, err
}

func newBodyReader(reader io.ReadCloser, maxSize int, recordBody bool) *bodyReader {
	var body *bytes.Buffer
	if recordBody {
		body = new(bytes.Buffer)
	}

	return &bodyReader{
		ReadCloser: reader,
		body:       body,
		maxSize:    maxSize,
		bytes:      0,
	}
}

var _ WrapResponse = (*response)(nil)

type response struct {
	*http.Response
	body    *bytes.Buffer
	maxSize int
	bytes   int
}

func newResponse(res *http.Response, maxSize int, recordBody bool) *response {
	var body *bytes.Buffer
	if recordBody {
		body = bytes.NewBuffer(make([]byte, 0, 1024))
		// limit read response body
		if res.Body != nil {
			// Read up to maxSize+1 to check if body is larger than maxSize
			limit := int64(maxSize) + 1
			lr := io.LimitReader(res.Body, limit)
			_, err := body.ReadFrom(lr)
			if err != nil {
				slog.Error("failed to read response body", "error", err)
			}

			// Restore response body
			// We need to create a new reader that contains the bytes we read + the remaining bytes in res.Body
			// If we read less than limit, it means we reached EOF of the original body.
			// If we read limit, it means there might be more bytes in the original body.

			// Note: body contains what we read.
			// res.Body has been consumed by the amount we read.
			// We need to construct a new ReadCloser.

			// Make a copy of the read data for restoration before potentially truncating 'body'
			readBytes := make([]byte, body.Len())
			copy(readBytes, body.Bytes())

			res.Body = &struct {
				io.Reader
				io.Closer
			}{
				Reader: io.MultiReader(bytes.NewReader(readBytes), res.Body),
				Closer: res.Body,
			}

			// Truncate the log buffer if it exceeds maxSize
			if body.Len() > maxSize {
				body.Truncate(maxSize)
			}
		}
	}

	return &response{
		Response: res,
		body:     body,
		maxSize:  maxSize,
		bytes:    0,
	}
}

func (r *response) Header() http.Header {
	return r.Response.Header
}

func (r *response) Status() int {
	return r.Response.StatusCode
}

func (r *response) BytesWritten() int {
	return r.bytes
}

func (r *response) Body() []byte {
	return r.body.Bytes()
}

type noopResponse struct{}

func (n noopResponse) Header() http.Header { return http.Header{} }
func (n noopResponse) Status() int         { return 0 }
func (n noopResponse) BytesWritten() int   { return 0 }
func (n noopResponse) Body() []byte        { return nil }
