package sloghttp

import (
	"context"
	"log/slog"
)

type slogRecord struct {
	Level slog.Level
	Msg   string
	Attrs []slog.Attr
}

type captureHandler struct {
	records []slogRecord
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	var attrs []slog.Attr
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a)
		return true
	})
	h.records = append(h.records, slogRecord{
		Level: r.Level,
		Msg:   r.Message,
		Attrs: attrs,
	})
	return nil
}
func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &captureHandler{records: append(h.records, slogRecord{Attrs: attrs})}
}
func (h *captureHandler) WithGroup(name string) slog.Handler { return h }

func findAttr(attrs []slog.Attr, key string) (slog.Attr, bool) {
	for _, a := range attrs {
		if a.Key == key {
			return a, true
		}
	}
	return slog.Attr{}, false
}
