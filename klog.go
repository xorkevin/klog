package klog

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"golang.org/x/exp/slog"
)

type (
	// Level is a log level
	Level int
)

// Log levels
const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelNone
)

// LevelFromString creates a log level from a string
func LevelFromString(s string) Level {
	switch s {
	case "DEBUG":
		return LevelDebug
	case "INFO":
		return LevelInfo
	case "WARN":
		return LevelWarn
	case "ERROR":
		return LevelError
	case "NONE":
		return LevelNone
	default:
		return LevelInfo
	}
}

// String implements [fmt.Stringer]
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelNone:
		return "NONE"
	default:
		return "UNSET"
	}
}

const (
	eventInlineAttrsSize = 5
)

type (
	// Attr is a log attribute
	Attr struct {
		Key   string
		Value Value
	}

	// Value is an attribute value
	Value struct {
		svalue slog.Value
	}

	// Event is a log event
	Event struct {
		Level          Level
		Time           time.Time
		PC             uintptr
		Path           string
		Message        string
		Context        context.Context
		inlineAttrs    [eventInlineAttrsSize]Attr
		numInlineAttrs int
		attrs          []Attr
	}
)

// NewEvent creates a new log event
func NewEvent(level Level, t time.Time, pc uintptr, path string, msg string, ctx context.Context) Event {
	return Event{
		Level:   level,
		Time:    t,
		PC:      pc,
		Path:    path,
		Message: msg,
		Context: ctx,
	}
}

// AddAttrs adds [Attr] to an event
func (e *Event) AddAttrs(attrs ...Attr) {
	n := copy(e.inlineAttrs[e.numInlineAttrs:], attrs)
	e.numInlineAttrs += n
	e.attrs = append(e.attrs, attrs[n:]...)
}

type (
	// Logger writes logs with context
	Logger interface {
		Enabled(level Level) bool
		Log(ctx context.Context, level Level, skip int, msg string, attrs ...Attr)
	}

	// SubLogger is a logger that can create subloggers
	SubLogger interface {
		Logger
		Sublogger(pathSegment string, attrs ...Attr) Logger
	}

	// Handler is a log event handler
	Handler interface {
		Handle(e Event)
		WithAttrs(attrs []Attr) Handler
	}

	// KLogger is a context logger that writes logs to a [Handler]
	KLogger struct {
		handler       Handler
		minLevel      Level
		clock         Clock
		pathSegment   string
		pathSeparator string
		parent        *KLogger
	}

	// Clock returns the current and monotonic time
	Clock interface {
		Time() time.Time
	}

	// LoggerOpt is an options function for [New]
	LoggerOpt = func(l *KLogger)

	// Frame is a logger caller frame
	Frame struct {
		Function string
		File     string
		Line     int
		PC       uintptr
	}
)

// New creates a new [Logger]
func New(opts ...LoggerOpt) Logger {
	l := &KLogger{
		minLevel:      LevelInfo,
		handler:       NewJSONSerializer(NewSyncWriter(os.Stdout)),
		clock:         RealTime{},
		pathSegment:   "",
		pathSeparator: "::",
		parent:        nil,
	}
	for _, i := range opts {
		i(l)
	}
	return l
}

// OptMinLevel returns a [LoggerOpt] that sets [KLogger] minLevel
func OptMinLevel(level Level) LoggerOpt {
	return func(l *KLogger) {
		l.minLevel = level
	}
}

// OptMinLevelStr returns a [LoggerOpt] that sets [KLogger] minLevel from a string
func OptMinLevelStr(level string) LoggerOpt {
	return OptMinLevel(LevelFromString(level))
}

// OptHandler returns a [LoggerOpt] that sets [KLogger] handler
func OptHandler(h Handler) LoggerOpt {
	return func(l *KLogger) {
		l.handler = h
	}
}

// OptClock returns a [LoggerOpt] that sets [KLogger] clock
func OptClock(c Clock) LoggerOpt {
	return func(l *KLogger) {
		l.clock = c
	}
}

// OptPath returns a [LoggerOpt] that sets [KLogger] path
func OptPath(segment string) LoggerOpt {
	return func(l *KLogger) {
		l.pathSegment = segment
	}
}

// OptPathSeparator returns a [LoggerOpt] that sets [KLogger] pathSeparator
func OptPathSeparator(separator string) LoggerOpt {
	return func(l *KLogger) {
		l.pathSeparator = separator
	}
}

// OptAttrs returns a [LoggerOpt] that sets [KLogger] attrs
func OptAttrs(attrs ...Attr) LoggerOpt {
	return func(l *KLogger) {
		l.handler = l.handler.WithAttrs(attrs)
	}
}

// Enabled implements [Logger] and returns if the logger is enabled for a level
func (l *KLogger) Enabled(level Level) bool {
	return level >= l.minLevel
}

// Log implements [Logger] and logs an event to its handler
func (l *KLogger) Log(ctx context.Context, level Level, skip int, msg string, attrs ...Attr) {
	if !l.Enabled(level) {
		return
	}

	t := l.clock.Time() // monotonic time
	pc := linepc(1 + skip)
	var fullpath strings.Builder
	l.buildPath(&fullpath)

	ev := NewEvent(level, t, pc, fullpath.String(), msg, ctx)
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

func (l *KLogger) buildPath(w io.Writer) {
	if l.parent != nil {
		l.parent.buildPath(w)
	}
	if l.pathSegment != "" {
		io.WriteString(w, l.pathSeparator)
		io.WriteString(w, l.pathSegment)
	}
}

func mergeAttrs(dest, from []Attr, seen map[string]struct{}) []Attr {
	for _, i := range from {
		if _, ok := seen[i.Key]; ok {
			continue
		}
		dest = append(dest, i)
		seen[i.Key] = struct{}{}
	}
	return dest
}

func linecaller(skip int) *Frame {
	callers := [1]uintptr{}
	if n := runtime.Callers(1+skip, callers[:]); n < 1 {
		return nil
	}
	frame, _ := runtime.CallersFrames(callers[:]).Next()
	return &Frame{
		Function: frame.Function,
		File:     frame.File,
		Line:     frame.Line,
		PC:       frame.PC,
	}
}

func (f Frame) String() string {
	return fmt.Sprintf("%s %s:%d", f.Function, f.File, f.Line)
}

// Sublogger implements [SubLogger] and creates a new sublogger
func (l *KLogger) Sublogger(path string, fields Fields) Logger {
	return &KLogger{
		minLevel: l.minLevel,
		handler:  l.handler,
		clock:    l.clock,
		path:     path,
		fields:   fields,
		parent:   l,
	}
}

type (
	subLogger struct {
		log    Logger
		path   string
		fields Fields
	}
)

// Sub returns a sublogger with path and fields.
//
// If l implements [SubLogger], then Sub returns l.Sublogger(path, fields),
// else a new Logger will be returned with a subppath of path.
func Sub(l Logger, path string, fields Fields) Logger {
	if sl, ok := l.(SubLogger); ok {
		return sl.Sublogger(path, fields)
	}
	return &subLogger{
		log:    l,
		path:   path,
		fields: fields,
	}
}

// Log implements [Logger]
func (l *subLogger) Log(ctx context.Context, level Level, path string, skip int, msg string, fields Fields) {
	allFields := Fields{}
	mergeFields(allFields, fields)
	for f := getCtxFields(ctx); f != nil; f = f.parent {
		mergeFields(allFields, f.fields)
	}
	mergeFields(allFields, l.fields)
	fullpath := l.path
	if path != "" {
		fullpath += "." + path
	}
	l.log.Log(context.Background(), level, fullpath, 1+skip, msg, allFields)
}

type (
	ctxKeyAttrs struct{}

	ctxAttrs struct {
		inlineAttrs    [eventInlineAttrsSize]Attr
		numInlineAttrs int
		attrs          []Attr
		parent         *ctxAttrs
	}
)

func (c *ctxAttrs) addAttrs(attrs ...Attr) {
	n := copy(c.inlineAttrs[c.numInlineAttrs:], attrs)
	c.numInlineAttrs += n
	c.attrs = append(c.attrs, attrs[n:]...)
}

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

// WithAttrs adds log attrs to context
func WithAttrs(ctx context.Context, attrs ...Attr) context.Context {
	return ExtendCtx(ctx, ctx, attrs...)
}

// ExtendCtx adds log attrs to context
func ExtendCtx(dest, ctx context.Context, attrs ...Attr) context.Context {
	k := &ctxAttrs{
		parent: getCtxAttrs(ctx),
	}
	k.addAttrs(attrs...)
	return setCtxAttrs(dest, k)
}
