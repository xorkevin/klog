package klog

import (
	"context"
	"os"
	"runtime"
	"time"
)

type (
	// Attr is a log attribute
	Attr struct {
		Key   string
		Value Value
	}
)

const (
	eventInlineAttrsSize = 5
)

type (
	// Event is a log event
	Event struct {
		Level   Level
		Time    time.Time
		PC      uintptr
		Message string
		Context context.Context
		attrs   attrsList
	}

	attrsList struct {
		inlineAttrs    [eventInlineAttrsSize]Attr
		numInlineAttrs int
		attrs          []Attr
	}
)

// NewEvent creates a new log event
func NewEvent(level Level, t time.Time, pc uintptr, msg string, ctx context.Context) Event {
	return Event{
		Level:   level,
		Time:    t,
		PC:      pc,
		Message: msg,
		Context: ctx,
	}
}

// AddAttrs adds [Attr] to an event
func (e *Event) AddAttrs(attrs ...Attr) {
	e.attrs.addAttrs(attrs)
}

func (a *attrsList) addAttrs(attrs []Attr) {
	n := copy(a.inlineAttrs[a.numInlineAttrs:], attrs)
	a.numInlineAttrs += n
	a.attrs = append(a.attrs, attrs[n:]...)
}

type (
	// Logger writes logs with context
	Logger interface {
		Enabled(level Level) bool
		Log(ctx context.Context, level Level, skip int, msg string, attrs ...Attr)
		Handler() Handler
	}

	// SubLogger is a logger that can create subloggers
	SubLogger interface {
		Logger
		Sublogger(pathSegment string, attrs []Attr) Logger
	}

	// Handler is a log event handler
	Handler interface {
		Enabled(level Level) bool
		Handle(e Event)
		Subhandler(pathSegment string, attrs []Attr) Handler
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
	defaultHandler = NewJSONSlogHandler(NewSyncWriter(os.Stdout))
	defaultLogger  = New()
)

// New creates a new [Logger]
func New(opts ...LoggerOpt) Logger {
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

// OptMinLevel returns a [LoggerOpt] that sets [KLogger] minLevel
func OptMinLevel(level Level) LoggerOpt {
	return func(l *KLogger) {
		l.minLevel = level
	}
}

// OptMinLevelStr returns a [LoggerOpt] that sets [KLogger] minLevel from a string
func OptMinLevelStr(s string) LoggerOpt {
	var level Level
	_ = level.UnmarshalText([]byte(s))
	return OptMinLevel(level)
}

// OptClock returns a [LoggerOpt] that sets [KLogger] clock
func OptClock(c Clock) LoggerOpt {
	return func(l *KLogger) {
		l.clock = c
	}
}

// OptSubhandler returns a [LoggerOpt] that sets [KLogger] handler sublogger
func OptSubhandler(pathSegment string, attrs []Attr) LoggerOpt {
	return func(l *KLogger) {
		l.handler = l.handler.Subhandler(pathSegment, attrs)
	}
}

// Enabled implements [Logger] and returns if the logger is enabled for a level
func (l *KLogger) Enabled(level Level) bool {
	return level >= l.minLevel && l.handler.Enabled(level)
}

// Log implements [Logger] and logs an event to its handler
func (l *KLogger) Log(ctx context.Context, level Level, skip int, msg string, attrs ...Attr) {
	if !l.Enabled(level) {
		return
	}

	t := l.clock.Time() // monotonic time
	pc := linepc(1 + skip)

	ev := NewEvent(level, t, pc, msg, ctx)
	ev.AddAttrs(attrs...)

	l.handler.Handle(ev)
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
func (l *KLogger) Sublogger(pathSegment string, attrs []Attr) Logger {
	return &KLogger{
		handler:  l.handler.Subhandler(pathSegment, attrs),
		minLevel: l.minLevel,
		clock:    l.clock,
	}
}

// Sub returns a sublogger with path and fields.
//
// If l implements [SubLogger], then Sub returns l.Sublogger(path, attrs),
// else a new Logger will be returned with a subppath of pathSegment.
func Sub(l Logger, pathSegment string, attrs ...Attr) Logger {
	if sl, ok := l.(SubLogger); ok {
		return sl.Sublogger(pathSegment, attrs)
	}
	return New(OptHandler(l.Handler()), OptSubhandler(pathSegment, attrs))
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
