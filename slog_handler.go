package klog

import (
	"context"
	"io"

	"golang.org/x/exp/slog"
)

var slogBuiltinKeys = map[string]struct{}{
	slog.LevelKey:   {},
	slog.TimeKey:    {},
	slog.SourceKey:  {},
	slog.MessageKey: {},
}

type (
	// SlogHandler writes logs to an [slog.Handler]
	SlogHandler struct {
		FieldTimeInfo string
		FieldCaller   string
		FieldPath     string
		FieldMsg      string
		PathSeparator string
		Path          string
		fieldsSet     map[string]struct{}
		slogHandler   slog.Handler
	}
)

// NewSlogHandler creates a new [*SlogHandler]
func NewSlogHandler(handler slog.Handler) *SlogHandler {
	return &SlogHandler{
		FieldTimeInfo: "ti",
		FieldCaller:   "caller",
		FieldPath:     "path",
		FieldMsg:      "msg",
		PathSeparator: ".",
		Path:          "",
		fieldsSet:     map[string]struct{}{},
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

func copyFieldsSet(s map[string]struct{}) map[string]struct{} {
	m := map[string]struct{}{}
	for k := range s {
		m[k] = struct{}{}
	}
	return m
}

func (h *SlogHandler) clone() *SlogHandler {
	return &SlogHandler{
		FieldTimeInfo: h.FieldTimeInfo,
		FieldCaller:   h.FieldCaller,
		FieldPath:     h.FieldPath,
		FieldMsg:      h.FieldMsg,
		PathSeparator: h.PathSeparator,
		Path:          h.Path,
		fieldsSet:     copyFieldsSet(h.fieldsSet),
		slogHandler:   h.slogHandler,
	}
}

func (h *SlogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.slogHandler.Enabled(ctx, level)
}

func (h *SlogHandler) Handle(ctx context.Context, rec slog.Record) {
	// TODO add attrs
	h.slogHandler.Handle(ctx, rec)
}

func (h *SlogHandler) Subhandler(pathSegment string, attrs []slog.Attr) Handler {
	h2 := h.clone()
	if pathSegment != "" {
		h2.Path += h2.PathSeparator + pathSegment
	}
	if len(attrs) > 0 {
		attrsToAdd := make([]slog.Attr, 0, len(attrs))
		for _, i := range attrs {
			if _, ok := slogBuiltinKeys[i.Key]; ok {
				continue
			}
			if _, ok := h2.fieldsSet[i.Key]; ok {
				continue
			}
			h2.fieldsSet[i.Key] = struct{}{}
			attrsToAdd = append(attrsToAdd, i)
		}
		if len(attrsToAdd) > 0 {
			h2.slogHandler = h2.slogHandler.WithAttrs(attrsToAdd)
		}
	}
	return h2
}
