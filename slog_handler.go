package klog

import (
	"context"
	"io"
	"time"

	"golang.org/x/exp/slog"
)

var slogBuiltinKeys = map[string]struct{}{
	slog.LevelKey:   {},
	slog.MessageKey: {},
}

type (
	// SlogHandler writes logs to an [slog.Handler]
	SlogHandler struct {
		FieldTimeInfo string
		FieldCaller   string
		FieldPath     string
		PathSeparator string
		Path          string
		attrKeySet    map[string]struct{}
		slogHandler   slog.Handler
	}
)

// NewSlogHandler creates a new [*SlogHandler]
func NewSlogHandler(handler slog.Handler) *SlogHandler {
	return &SlogHandler{
		FieldTimeInfo: "t",
		FieldCaller:   "caller",
		FieldPath:     "path",
		PathSeparator: ".",
		Path:          "",
		attrKeySet:    map[string]struct{}{},
		slogHandler:   handler,
	}
}

func NewJSONSlogHandler(w io.Writer) *SlogHandler {
	return NewSlogHandler(
		slog.HandlerOptions{
			Level: slog.LevelDebug,
		}.NewJSONHandler(w),
	)
}

func copyStringSet(s map[string]struct{}) map[string]struct{} {
	m := map[string]struct{}{}
	for k := range s {
		m[k] = struct{}{}
	}
	return m
}

func (h *SlogHandler) clone() *SlogHandler {
	return &SlogHandler{
		FieldTimeInfo: h.FieldTimeInfo,
		FieldPath:     h.FieldPath,
		PathSeparator: h.PathSeparator,
		Path:          h.Path,
		attrKeySet:    copyStringSet(h.attrKeySet),
		slogHandler:   h.slogHandler,
	}
}

func (h *SlogHandler) checkAttrKey(k string) bool {
	if k == h.FieldTimeInfo || k == h.FieldCaller || k == h.FieldPath {
		return true
	}
	if _, ok := slogBuiltinKeys[k]; ok {
		return true
	}
	if _, ok := h.attrKeySet[k]; ok {
		return true
	}
	return false
}

func (h *SlogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.slogHandler.Enabled(ctx, level)
}

func (h *SlogHandler) Handle(ctx context.Context, r slog.Record) {
	r2 := slog.NewRecord(time.Time{}, r.Level, r.Message, 0)
	// TODO add attrs
	h.slogHandler.Handle(ctx, r2)
}

func (h *SlogHandler) Subhandler(pathSegment string, attrs []slog.Attr) Handler {
	h2 := h.clone()
	if pathSegment != "" {
		h2.Path += h2.PathSeparator + pathSegment
	}
	if len(attrs) > 0 {
		attrsToAdd := make([]slog.Attr, 0, len(attrs))
		for _, i := range attrs {
			if h2.checkAttrKey(i.Key) {
				continue
			}
			h2.attrKeySet[i.Key] = struct{}{}
			attrsToAdd = append(attrsToAdd, i)
		}
		if len(attrsToAdd) > 0 {
			h2.slogHandler = h2.slogHandler.WithAttrs(attrsToAdd)
		}
	}
	return h2
}
