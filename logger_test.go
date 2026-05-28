package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseLog parses a single JSON log line into a map.
func parseLog(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var m map[string]any
	err := json.Unmarshal(data, &m)
	require.NoError(t, err)
	return m
}

func TestNew_DefaultHandler(t *testing.T) {
	// New() with no args should create a default stdout handler at Info level.
	l := New()
	assert.NotNil(t, l)
}

func TestNew_SingleHandler(t *testing.T) {
	var buf bytes.Buffer
	l := New(NewJSONHandler(&buf, WithLevel(slog.LevelInfo)))

	l.Info("hello", slog.String("key", "value"))
	assert.NotEmpty(t, buf.Bytes())

	m := parseLog(t, buf.Bytes())
	assert.Equal(t, "hello", m["msg"])
	assert.Equal(t, "INFO", m["level"])
	assert.Equal(t, "value", m["key"])
}

func TestNew_MultiHandler_LevelRouting(t *testing.T) {
	var debugBuf, infoBuf bytes.Buffer

	l := New(
		NewJSONHandler(&debugBuf, WithLevel(slog.LevelDebug)),
		NewJSONHandler(&infoBuf, WithLevel(slog.LevelInfo)),
	)

	// Debug → only debugBuf
	l.Debug("debug message")
	assert.NotEmpty(t, debugBuf.Bytes())
	assert.Empty(t, infoBuf.Bytes())

	debugBuf.Reset()
	infoBuf.Reset()

	// Info → both handlers
	l.Info("info message")
	assert.NotEmpty(t, debugBuf.Bytes())
	assert.NotEmpty(t, infoBuf.Bytes())

	debugBuf.Reset()
	infoBuf.Reset()

	// Warn → both handlers
	l.Warn("warn message")
	assert.NotEmpty(t, debugBuf.Bytes())
	assert.NotEmpty(t, infoBuf.Bytes())

	debugBuf.Reset()
	infoBuf.Reset()

	// Error → both handlers
	l.Error("error message")
	assert.NotEmpty(t, debugBuf.Bytes())
	assert.NotEmpty(t, infoBuf.Bytes())
}

func TestNewJSONHandler_TimeFormat(t *testing.T) {
	var buf bytes.Buffer
	l := New(NewJSONHandler(&buf))

	l.Info("test")

	m := parseLog(t, buf.Bytes())
	ts, ok := m["time"].(string)
	require.True(t, ok)
	_, err := time.Parse("2006-01-02 15:04:05.000", ts)
	assert.NoError(t, err)
}

func TestNewJSONHandler_NoSource(t *testing.T) {
	var buf bytes.Buffer
	l := New(NewJSONHandler(&buf))

	l.Info("test")

	m := parseLog(t, buf.Bytes())
	_, hasSource := m["source"]
	assert.False(t, hasSource)
}

func TestNewJSONHandler_WithSource(t *testing.T) {
	var buf bytes.Buffer
	l := New(NewJSONHandler(&buf, WithSource()))

	l.Info("test")

	m := parseLog(t, buf.Bytes())
	_, hasSource := m["source"]
	assert.True(t, hasSource)
}

func TestNewJSONHandler_WithLevel(t *testing.T) {
	var buf bytes.Buffer
	l := New(NewJSONHandler(&buf, WithLevel(slog.LevelWarn)))

	l.Info("should not appear")
	assert.Empty(t, buf.Bytes())

	l.Warn("should appear")
	assert.NotEmpty(t, buf.Bytes())

	m := parseLog(t, buf.Bytes())
	assert.Equal(t, "should appear", m["msg"])
}

func TestNewJSONHandler_WithReplaceAttr(t *testing.T) {
	var buf bytes.Buffer
	l := New(NewJSONHandler(&buf, WithReplaceAttr(func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == "msg" {
			return slog.String("msg", "replaced: "+a.Value.String())
		}
		return a
	})))

	l.Info("original")

	m := parseLog(t, buf.Bytes())
	assert.Equal(t, "replaced: original", m["msg"])
}

func TestNewFileWriter(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "test.log")

	w := NewFileWriter(filename, WithMaxSize(1024*1024), WithMaxFiles(2), WithMaxAge(1))
	defer w.Close()

	l := New(NewJSONHandler(w, WithLevel(slog.LevelInfo)))
	l.Info("file test", slog.String("key", "value"))

	data, err := os.ReadFile(filename)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	m := parseLog(t, data)
	assert.Equal(t, "file test", m["msg"])
	assert.Equal(t, "value", m["key"])
}

func TestNewFileWriter_Defaults(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "defaults.log")

	// Test with default options (no FileOption)
	w := NewFileWriter(filename)
	defer w.Close()

	_, err := w.Write([]byte("hello"))
	require.NoError(t, err)

	data, err := os.ReadFile(filename)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))
}

func TestDefault_Logger(t *testing.T) {
	assert.NotNil(t, Default())
}

func TestSetDefault(t *testing.T) {
	var buf bytes.Buffer
	l := New(NewJSONHandler(&buf, WithLevel(slog.LevelDebug)))
	original := Default()
	SetDefault(l)
	defer SetDefault(original)

	Debug("default test")
	assert.NotEmpty(t, buf.Bytes())

	m := parseLog(t, buf.Bytes())
	assert.Equal(t, "default test", m["msg"])
}

func TestNewJSONHandler_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	l := New(NewJSONHandler(&buf, WithLevel(slog.LevelWarn)))

	l.Info("should not appear")
	assert.Empty(t, buf.Bytes())

	l.Warn("should appear")
	assert.NotEmpty(t, buf.Bytes())

	m := parseLog(t, buf.Bytes())
	assert.Equal(t, "should appear", m["msg"])
}

func TestMultiHandler_WithAttrs(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	l := New(
		NewJSONHandler(&buf1, WithLevel(slog.LevelInfo)),
		NewJSONHandler(&buf2, WithLevel(slog.LevelInfo)),
	)

	child := l.With(slog.String("shared", "attr"))
	child.Info("with attrs")

	m1 := parseLog(t, buf1.Bytes())
	assert.Equal(t, "attr", m1["shared"])

	m2 := parseLog(t, buf2.Bytes())
	assert.Equal(t, "attr", m2["shared"])
}

func TestMultiHandler_WithGroup(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	l := New(
		NewJSONHandler(&buf1, WithLevel(slog.LevelInfo)),
		NewJSONHandler(&buf2, WithLevel(slog.LevelInfo)),
	)

	child := l.WithGroup("request")
	child.Info("with group", slog.String("path", "/api"))

	m1 := parseLog(t, buf1.Bytes())
	req1, ok := m1["request"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "/api", req1["path"])

	m2 := parseLog(t, buf2.Bytes())
	req2, ok := m2["request"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "/api", req2["path"])
}
