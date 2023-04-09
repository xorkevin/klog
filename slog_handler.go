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
		FieldMod      string
		ModSeparator  string
		Mod           string
		attrKeySet    map[string]struct{}
		slogHandler   slog.Handler
	}
)

// NewSlogHandler creates a new [*SlogHandler]
func NewSlogHandler(handler slog.Handler) *SlogHandler {
	return &SlogHandler{
		FieldTimeInfo: "t",
		FieldCaller:   "caller",
		FieldMod:      "mod",
		ModSeparator:  ".",
		Mod:           "",
		attrKeySet:    map[string]struct{}{},
		slogHandler:   handler,
	}
}

func NewTextSlogHandler(w io.Writer) *SlogHandler {
	return NewSlogHandler(
		slog.HandlerOptions{
			Level: LevelDebug,
		}.NewTextHandler(w),
	)
}

func NewJSONSlogHandler(w io.Writer) *SlogHandler {
	return NewSlogHandler(
		slog.HandlerOptions{
			Level: LevelDebug,
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
		FieldMod:      h.FieldMod,
		ModSeparator:  h.ModSeparator,
		Mod:           h.Mod,
		attrKeySet:    copyStringSet(h.attrKeySet),
		slogHandler:   h.slogHandler,
	}
}

func (h *SlogHandler) checkAttrKey(k string) bool {
	if k == "" {
		return true
	}
	if k == h.FieldTimeInfo || k == h.FieldCaller || k == h.FieldMod {
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
	if h.FieldTimeInfo != "" && !r.Time.IsZero() {
		mt := r.Time
		t := mt.Round(0)
		r2.AddAttrs(
			AGroup(
				h.FieldTimeInfo,
				AInt64("mono_us", mt.UnixMicro()),
				AInt64("unix_us", t.UnixMicro()),
				AString("time", t.Format(time.RFC3339Nano)),
			),
		)
	}
	if h.FieldCaller != "" && r.PC != 0 {
		frame := linecaller(r.PC)
		r2.AddAttrs(
			AGroup(
				h.FieldCaller,
				AString("fn", frame.Function),
				AString("src", frame.File+":"+strconv.Itoa(frame.Line)),
			),
		)
	}
	if h.FieldMod != "" && h.Mod != "" {
		r2.AddAttrs(AString(h.FieldMod, h.Mod))
	}
	attrKeys := map[string]struct{}{}
	addFilteredAttrs := func(attr Attr) {
		if h.checkAttrKey(attr.Key) {
			return
		}
		if _, ok := attrKeys[attr.Key]; ok {
			return
		}
		attrKeys[attr.Key] = struct{}{}
		r2.AddAttrs(attr)
	}
	r.Attrs(addFilteredAttrs)
	for ctxAttrs := getCtxAttrs(ctx); ctxAttrs != nil; ctxAttrs = ctxAttrs.parent {
		ctxAttrs.attrs.readAttrs(addFilteredAttrs)
	}
	return h.slogHandler.Handle(ctx, r2)
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
