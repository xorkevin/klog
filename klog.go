package klog

import (
	"context"
	"os"
	"runtime"
	"time"

	"golang.org/x/exp/slog"
)

type (
	// Logger writes logs with context
	Logger interface {
		Enabled(ctx context.Context, level slog.Level) bool
		Log(ctx context.Context, level slog.Level, skip int, msg string, attrs ...slog.Attr)
		Handler() Handler
		Sublogger(pathSegment string, attrs []slog.Attr) Logger
	}

	// Handler is a log event handler
	Handler interface {
		Enabled(ctx context.Context, level slog.Level) bool
		Handle(ctx context.Context, rec slog.Record)
		Subhandler(pathSegment string, attrs []slog.Attr) Handler
	}

	// KLogger is a context logger that writes logs to a [Handler]
	KLogger struct {
		handler  Handler
		minLevel slog.Level
		clock    Clock
	}

	// Clock returns the current and monotonic time
	Clock interface {
		Time() time.Time
	}

	// LoggerOpt is an options function for [New]
	LoggerOpt = func(l *KLogger)
)

var (
	defaultHandler Handler = NewJSONSlogHandler(NewSyncWriter(os.Stdout))
	defaultLogger  Logger  = New()
)

// New creates a new [Logger]
func New(opts ...LoggerOpt) *KLogger {
	l := &KLogger{
		handler:  defaultHandler,
		minLevel: slog.LevelInfo,
		clock:    RealTime{},
	}
	for _, i := range opts {
		i(l)
	}
	return l
}

// OptHandler returns a [LoggerOpt] that sets [KLogger] handler
func OptHandler(h Handler) LoggerOpt {
	return func(l *KLogger) {
		l.handler = h
	}
}

// OptMinLevel returns a [LoggerOpt] that sets [KLogger] minLevel
func OptMinLevel(level slog.Level) LoggerOpt {
	return func(l *KLogger) {
		l.minLevel = level
	}
}

// OptMinLevelStr returns a [LoggerOpt] that sets [KLogger] minLevel from a string
func OptMinLevelStr(s string) LoggerOpt {
	var level slog.Level
	if err := level.UnmarshalText([]byte(s)); err != nil {
		level = slog.LevelInfo
	}
	return OptMinLevel(level)
}

// OptClock returns a [LoggerOpt] that sets [KLogger] clock
func OptClock(c Clock) LoggerOpt {
	return func(l *KLogger) {
		l.clock = c
	}
}

// Enabled implements [Logger] and returns if the logger is enabled for a level
func (l *KLogger) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= l.minLevel && l.handler.Enabled(ctx, level)
}

// Log implements [Logger] and logs an event to its handler
func (l *KLogger) Log(ctx context.Context, level slog.Level, skip int, msg string, attrs ...slog.Attr) {
	if !l.Enabled(ctx, level) {
		return
	}

	t := l.clock.Time() // monotonic time
	pc := linepc(1 + skip)

	rec := slog.NewRecord(t, level, msg, pc)
	rec.AddAttrs(attrs...)

	l.handler.Handle(ctx, rec)
}

func linepc(skip int) uintptr {
	var callers [1]uintptr
	if n := runtime.Callers(2+skip, callers[:]); n < 1 {
		return 0
	}
	return callers[0]
}

// Handler implements [Logger] and returns the handler
func (l *KLogger) Handler() Handler {
	return l.handler
}

// Sublogger implements [SubLogger] and creates a new sublogger
func (l *KLogger) Sublogger(pathSegment string, attrs []slog.Attr) Logger {
	return &KLogger{
		handler:  l.handler.Subhandler(pathSegment, attrs),
		minLevel: l.minLevel,
		clock:    l.clock,
	}
}

const (
	eventInlineAttrsSize = 5
)

type (
	attrsList struct {
		inlineAttrs    [eventInlineAttrsSize]slog.Attr
		numInlineAttrs int
		attrs          []slog.Attr
	}
)

func (a *attrsList) addAttrs(attrs []slog.Attr) {
	n := copy(a.inlineAttrs[a.numInlineAttrs:], attrs)
	a.numInlineAttrs += n
	a.attrs = append(a.attrs, attrs[n:]...)
}

type (
	ctxKeyAttrs struct{}

	ctxAttrs struct {
		attrs  attrsList
		parent *ctxAttrs
	}
)

func getCtxAttrs(ctx context.Context) *ctxAttrs {
	if ctx == nil {
		return nil
	}
	v := ctx.Value(ctxKeyAttrs{})
	if v == nil {
		return nil
	}
	return v.(*ctxAttrs)
}

func setCtxAttrs(ctx context.Context, fields *ctxAttrs) context.Context {
	return context.WithValue(ctx, ctxKeyAttrs{}, fields)
}

// CtxWithAttrs adds log attrs to context
func CtxWithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	return ExtendCtx(ctx, ctx, attrs...)
}

// ExtendCtx adds log attrs to context
func ExtendCtx(dest, ctx context.Context, attrs ...slog.Attr) context.Context {
	k := &ctxAttrs{
		parent: getCtxAttrs(ctx),
	}
	k.attrs.addAttrs(attrs)
	return setCtxAttrs(dest, k)
}
