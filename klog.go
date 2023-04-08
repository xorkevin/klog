package klog

import (
	"context"
	"os"
	"runtime"
	"time"

	"golang.org/x/exp/slog"
)

type (
	Level  = slog.Level
	Attr   = slog.Attr
	Record = slog.Record
)

const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

func AString(key string, value string) Attr {
	return slog.String(key, value)
}

func AInt(key string, value int) Attr {
	return slog.Int(key, value)
}

func AInt64(key string, value int64) Attr {
	return slog.Int64(key, value)
}

func AUint64(key string, value uint64) Attr {
	return slog.Uint64(key, value)
}

func AFloat64(key string, value float64) Attr {
	return slog.Float64(key, value)
}

func ABool(key string, value bool) Attr {
	return slog.Bool(key, value)
}

func ATime(key string, value time.Time) Attr {
	return slog.Time(key, value)
}

func ADuration(key string, value time.Duration) Attr {
	return slog.Duration(key, value)
}

func AGroup(key string, attrs ...Attr) Attr {
	return slog.Group(key, attrs...)
}

func AAny(key string, value any) Attr {
	return slog.Any(key, value)
}

func NewRecord(t time.Time, level Level, msg string, pc uintptr) Record {
	return slog.NewRecord(t, level, msg, pc)
}

type (
	// Logger writes logs with context
	Logger interface {
		Enabled(ctx context.Context, level Level) bool
		Log(ctx context.Context, level Level, skip int, msg string, attrs ...Attr)
		Handler() Handler
		Sublogger(modSegment string, attrs ...Attr) Logger
	}

	// Handler is a log event handler
	Handler interface {
		Enabled(ctx context.Context, level Level) bool
		Handle(ctx context.Context, rec Record) error
		Subhandler(modSegment string, attrs []Attr) Handler
	}

	// KLogger is a context logger that writes logs to a [Handler]
	KLogger struct {
		handler  Handler
		minLevel Level
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
	defaultHandler Handler = NewJSONSlogHandler(NewSyncWriter(os.Stderr))
	defaultLogger  Logger  = New()
)

// New creates a new [Logger]
func New(opts ...LoggerOpt) *KLogger {
	l := &KLogger{
		handler:  defaultHandler,
		minLevel: LevelInfo,
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

// OptSubhandler returns a [LoggerOpt] that sets [KLogger] handler
func OptSubhandler(modSegment string, attrs ...Attr) LoggerOpt {
	return func(l *KLogger) {
		l.handler = l.handler.Subhandler(modSegment, attrs)
	}
}

// OptMinLevel returns a [LoggerOpt] that sets [KLogger] minLevel
func OptMinLevel(level Level) LoggerOpt {
	return func(l *KLogger) {
		l.minLevel = level
	}
}

// OptMinLevelStr returns a [LoggerOpt] that sets [KLogger] minLevel from a string
func OptMinLevelStr(s string) LoggerOpt {
	var level Level
	if err := level.UnmarshalText([]byte(s)); err != nil {
		level = LevelInfo
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
func (l *KLogger) Enabled(ctx context.Context, level Level) bool {
	return level >= l.minLevel && l.handler.Enabled(ctx, level)
}

// Log implements [Logger] and logs an event to its handler
func (l *KLogger) Log(ctx context.Context, level Level, skip int, msg string, attrs ...Attr) {
	if !l.Enabled(ctx, level) {
		return
	}

	t := l.clock.Time() // monotonic time
	pc := linepc(1 + skip)

	rec := slog.NewRecord(t, level, msg, pc)
	rec.AddAttrs(attrs...)

	// ignore errors for failing to handle logs
	_ = l.handler.Handle(ctx, rec)
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
func (l *KLogger) Sublogger(modSegment string, attrs ...Attr) Logger {
	return &KLogger{
		handler:  l.handler.Subhandler(modSegment, attrs),
		minLevel: l.minLevel,
		clock:    l.clock,
	}
}

const (
	eventInlineAttrsSize = 5
)

type (
	attrsList struct {
		inlineAttrs    [eventInlineAttrsSize]Attr
		numInlineAttrs int
		attrs          []Attr
	}
)

func (a *attrsList) addAttrs(attrs []Attr) {
	n := copy(a.inlineAttrs[a.numInlineAttrs:], attrs)
	a.numInlineAttrs += n
	a.attrs = append(a.attrs, attrs[n:]...)
}

func (l *attrsList) readAttrs(f func(a Attr)) {
	for i := 0; i < l.numInlineAttrs; i++ {
		f(l.inlineAttrs[i])
	}
	for _, i := range l.attrs {
		f(i)
	}
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
func CtxWithAttrs(ctx context.Context, attrs ...Attr) context.Context {
	return ExtendCtx(ctx, ctx, attrs...)
}

// ExtendCtx adds log attrs to context
func ExtendCtx(dest, ctx context.Context, attrs ...Attr) context.Context {
	k := &ctxAttrs{
		parent: getCtxAttrs(ctx),
	}
	k.attrs.addAttrs(attrs)
	return setCtxAttrs(dest, k)
}
