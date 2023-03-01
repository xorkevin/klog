package klog

import (
	"context"
	"io"
	"strconv"
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

func NewTextSlogHandler(w io.Writer) *SlogHandler {
	return NewSlogHandler(
		slog.HandlerOptions{
			Level: slog.LevelDebug,
		}.NewTextHandler(w),
	)
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
		FieldCaller:   h.FieldCaller,
		FieldPath:     h.FieldPath,
		PathSeparator: h.PathSeparator,
		Path:          h.Path,
		attrKeySet:    copyStringSet(h.attrKeySet),
		slogHandler:   h.slogHandler,
	}
}

func (h *SlogHandler) checkAttrKey(k string) bool {
	if k == "" {
		return true
	}
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
	if !r.Time.IsZero() {
		mt := r.Time
		t := mt.Round(0)
		r2.AddAttrs(
			slog.Group(
				h.FieldTimeInfo,
				slog.Int64("mono_us", mt.UnixMicro()),
				slog.Int64("unix_us", t.UnixMicro()),
				slog.String("time", t.Format(time.RFC3339Nano)),
			),
		)
	}
	if r.PC != 0 {
		frame := linecaller(r.PC)
		r2.AddAttrs(
			slog.Group(
				h.FieldCaller,
				slog.String("fn", frame.Function),
				slog.String("src", frame.File+":"+strconv.Itoa(frame.Line)),
			),
		)
	}
	if h.Path != "" {
		r2.AddAttrs(slog.String(h.FieldPath, h.Path))
	}
	attrKeys := map[string]struct{}{}
	r.Attrs(func(attr slog.Attr) {
		if h.checkAttrKey(attr.Key) {
			return
		}
		if _, ok := attrKeys[attr.Key]; ok {
			return
		}
		attrKeys[attr.Key] = struct{}{}
		r2.AddAttrs(attr)
	})
	for ctxAttrs := getCtxAttrs(ctx); ctxAttrs != nil; ctxAttrs = ctxAttrs.parent {
		ctxAttrs.attrs.readAttrs(func(attr slog.Attr) {
			if h.checkAttrKey(attr.Key) {
				return
			}
			if _, ok := attrKeys[attr.Key]; ok {
				return
			}
			attrKeys[attr.Key] = struct{}{}
			r2.AddAttrs(attr)
		})
	}
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
