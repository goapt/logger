package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLogger_ModeCustom(t *testing.T) {
	var buf bytes.Buffer
	l := New(&Config{Mode: ModeCustom, Writer: &buf})

	l.Info("hello", slog.String("foo", "bar"), slog.Int("x", 1))

	got := buf.Bytes()
	assert.NotEmpty(t, got)

	var m map[string]any
	err := json.Unmarshal(got, &m)
	assert.NoError(t, err)

	assert.Equal(t, "hello", m["msg"])
	assert.Equal(t, "INFO", m["level"])
	assert.Equal(t, "bar", m["foo"])

	xv, ok := m["x"].(float64)
	assert.True(t, ok)
	assert.Equal(t, 1.0, xv)

	ts, ok := m["time"].(string)
	assert.True(t, ok)
	_, err = time.Parse("2006-01-02 15:04:05.000", ts)
	assert.NoError(t, err)

	_, hasSource := m["source"]
	assert.False(t, hasSource)
}
