package klog

import (
	"context"
	"io"
	"log/slog"
	"maps"
	"strconv"
	"time"

	"xorkevin.dev/kerrors"
)

var slogBuiltinKeys = map[string]struct{}{
	slog.LevelKey:   {},
	slog.MessageKey: {},
}

type (
	// SlogHandler writes logs to an [slog.Handler]
	SlogHandler struct {
		FieldTime    string
		FieldTimeLoc *time.Location
		FieldSrc     string
		FieldMod     string
		ModSeparator string
		Mod          string
		attrKeySet   map[string]struct{}
		slogHandler  slog.Handler
	}
)

// NewSlogHandler creates a new [*SlogHandler]
func NewSlogHandler(handler slog.Handler) *SlogHandler {
	return &SlogHandler{
		FieldTime:    "t",
		FieldTimeLoc: time.UTC,
		FieldSrc:     "src",
		FieldMod:     "mod",
		ModSeparator: ".",
		Mod:          "",
		attrKeySet:   map[string]struct{}{},
		slogHandler:  handler,
	}
}

func NewTextSlogHandler(w io.Writer) *SlogHandler {
	return NewSlogHandler(
		slog.NewTextHandler(w,
			&slog.HandlerOptions{
				Level: LevelDebug,
			},
		),
	)
}

func NewJSONSlogHandler(w io.Writer) *SlogHandler {
	return NewSlogHandler(
		slog.NewJSONHandler(w,
			&slog.HandlerOptions{
				Level: LevelDebug,
			},
		),
	)
}

func (h *SlogHandler) clone() *SlogHandler {
	return &SlogHandler{
		FieldTime:    h.FieldTime,
		FieldTimeLoc: h.FieldTimeLoc,
		FieldSrc:     h.FieldSrc,
		FieldMod:     h.FieldMod,
		ModSeparator: h.ModSeparator,
		Mod:          h.Mod,
		attrKeySet:   maps.Clone(h.attrKeySet),
		slogHandler:  h.slogHandler,
	}
}

func (h *SlogHandler) checkAttrKey(k string) bool {
	if k == "" {
		return true
	}
	if k == h.FieldTime || k == h.FieldSrc || k == h.FieldMod {
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

func (h *SlogHandler) Enabled(ctx context.Context, level Level) bool {
	return h.slogHandler.Enabled(ctx, level)
}

func (h *SlogHandler) Handle(ctx context.Context, r Record) error {
	r2 := NewRecord(time.Time{}, r.Level, r.Message, 0)
	if h.FieldTime != "" && !r.Time.IsZero() {
		r2.AddAttrs(AString(h.FieldTime, r.Time.In(h.FieldTimeLoc).Format(time.RFC3339Nano)))
	}
	if h.FieldSrc != "" && r.PC != 0 {
		frame := linecaller(r.PC)
		r2.AddAttrs(
			AGroup(
				h.FieldSrc,
				AString("fn", frame.Function),
				AString("file", frame.File+":"+strconv.Itoa(frame.Line)),
			),
		)
	}
	if h.FieldMod != "" && h.Mod != "" {
		r2.AddAttrs(AString(h.FieldMod, h.Mod))
	}
	attrKeys := map[string]struct{}{}
	addFilteredAttrs := func(attr Attr) bool {
		if h.checkAttrKey(attr.Key) {
			return true
		}
		if _, ok := attrKeys[attr.Key]; ok {
			return true
		}
		attrKeys[attr.Key] = struct{}{}
		if attr.Value.Kind() == slog.KindAny {
			if verr, ok := attr.Value.Any().(error); ok {
				attr = AAny(attr.Key, errLogValuer{err: verr})
			}
		}
		r2.AddAttrs(attr)
		return true
	}
	// ctx attrs have precedence of child before parent adhering to
	// [context.Context] Value semantics
	for ctxAttrs := getCtxAttrs(ctx); ctxAttrs != nil; ctxAttrs = ctxAttrs.parent {
		ctxAttrs.attrs.readAttrs(addFilteredAttrs)
	}
	// attrs on the record have lowest precedence as to avoid overriding attrs on
	// the context and handler
	r.Attrs(addFilteredAttrs)
	return h.slogHandler.Handle(ctx, r2)
}

type (
	errLogValuer struct {
		err error
	}
)

func (e errLogValuer) LogValue() slog.Value {
	return slog.AnyValue(kerrors.JSONValue(e.err))
}

func (h *SlogHandler) Subhandler(modSegment string, attrs []Attr) Handler {
	h2 := h.clone()
	if modSegment != "" {
		h2.Mod += h2.ModSeparator + modSegment
	}
	if len(attrs) > 0 {
		attrsToAdd := make([]Attr, 0, len(attrs))
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
