package klog

import (
	"io"

	"golang.org/x/exp/slog"
)

var (
	slogBuiltinKeys = map[string]struct{}{
		slog.LevelKey:   {},
		slog.TimeKey:    {},
		slog.SourceKey:  {},
		slog.MessageKey: {},
	}
)

func levelToSlogLevel(level Level) slog.Level {
	switch level {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	case LevelNone:
		return slog.LevelError + 4
	default:
		return slog.LevelInfo
	}
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
		FieldsSet     map[string]struct{}
		SlogHandler   slog.Handler
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
		FieldsSet:     map[string]struct{}{},
		SlogHandler:   handler,
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
		FieldsSet:     copyFieldsSet(h.FieldsSet),
		SlogHandler:   h.SlogHandler,
	}
}

func (h *SlogHandler) Enabled(level Level) bool {
	return h.SlogHandler.Enabled(nil, levelToSlogLevel(level))
}

func (h *SlogHandler) Handle(e Event) {
	r := slog.NewRecord(e.Time, levelToSlogLevel(e.Level), e.Message, e.PC, e.Context)
	// TODO add attrs
	h.SlogHandler.Handle(r)
}

func (h *SlogHandler) Subhandler(pathSegment string, attrs []Attr) Handler {
	h2 := h.clone()
	if pathSegment != "" {
		h2.Path += h2.PathSeparator + pathSegment
	}
	if len(attrs) > 0 {
		attrsToAdd := make([]slog.Attr, 0, len(attrs))
		for _, i := range attrs {
			if _, ok := h2.FieldsSet[i.Key]; !ok {
				h2.FieldsSet[i.Key] = struct{}{}
				attrsToAdd = append(attrsToAdd, slog.Attr{
					Key:   i.Key,
					Value: i.Value.svalue,
				})
			}
		}
		if len(attrsToAdd) > 0 {
			h2.SlogHandler = h2.SlogHandler.WithAttrs(attrsToAdd)
		}
	}
	return h2
}
