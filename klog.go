package klog

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
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

type (
	// Logger writes logs with context
	Logger interface {
		Log(ctx context.Context, level Level, path string, skip int, msg string, fields Fields)
	}

	// SubLogger is a logger that can create subloggers
	SubLogger interface {
		Logger
		Sublogger(path string, fields Fields) Logger
	}

	// LoggerFn is a logger that can conditionally compute fields to log
	LoggerFn interface {
		Logger
		LogFn(ctx context.Context, level Level, path string, skip int, fn FieldsFunc)
	}

	// Fields is associated log data
	Fields map[string]interface{}

	// FieldsFunc returns a log message and fields
	FieldsFunc = func() (msg string, fields Fields)

	// Serializer is a log serializer adapter
	Serializer interface {
		Log(level Level, t, mt time.Time, caller *Frame, path string, msg string, fields Fields)
	}

	// Frame is a logger caller frame
	Frame struct {
		Function string
		File     string
		Line     int
		PC       uintptr
	}

	// KLogger is a context logger that writes logs to a [Serializer]
	KLogger struct {
		minLevel   Level
		serializer Serializer
		clock      Clock
		path       string
		fields     Fields
		parent     *KLogger
	}

	// Clock returns the current and monotonic time
	Clock interface {
		Time() (cur time.Time, mono time.Time)
	}

	// LoggerOpt is an options function for [New]
	LoggerOpt = func(l *KLogger)
)

// New creates a new [Logger]
func New(opts ...LoggerOpt) Logger {
	l := &KLogger{
		minLevel:   LevelInfo,
		serializer: NewJSONSerializer(NewSyncWriter(os.Stdout)),
		clock:      RealTime{},
		path:       "",
		fields:     nil,
		parent:     nil,
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

// OptSerializer returns a [LoggerOpt] that sets [KLogger] serializer
func OptSerializer(s Serializer) LoggerOpt {
	return func(l *KLogger) {
		l.serializer = s
	}
}

// OptClock returns a [LoggerOpt] that sets [KLogger] clock
func OptClock(c Clock) LoggerOpt {
	return func(l *KLogger) {
		l.clock = c
	}
}

// OptPath returns a [LoggerOpt] that sets [KLogger] path
func OptPath(path string) LoggerOpt {
	return func(l *KLogger) {
		l.path = path
	}
}

// OptFields returns a [LoggerOpt] that sets [KLogger] fields
func OptFields(fields Fields) LoggerOpt {
	return func(l *KLogger) {
		l.fields = fields
	}
}

func mergeFields(dest, from Fields) {
	for k, v := range from {
		if _, ok := dest[k]; !ok {
			dest[k] = v
		}
	}
}

func (l *KLogger) buildPath(s *strings.Builder) {
	if l.parent != nil {
		l.parent.buildPath(s)
	}
	if l.path != "" {
		s.WriteByte('.')
		s.WriteString(l.path)
	}
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

// Log implements [Logger]
func (l *KLogger) Log(ctx context.Context, level Level, path string, skip int, msg string, fields Fields) {
	if level < l.minLevel {
		return
	}

	t, mt := l.clock.Time()
	caller := linecaller(1 + skip)
	fullpath := strings.Builder{}
	l.buildPath(&fullpath)
	if path != "" {
		fullpath.WriteByte('.')
		fullpath.WriteString(path)
	}
	allFields := Fields{}
	mergeFields(allFields, fields)
	for f := getCtxFields(ctx); f != nil; f = f.parent {
		mergeFields(allFields, f.fields)
	}
	for k := l; k != nil; k = k.parent {
		mergeFields(allFields, k.fields)
	}
	l.serializer.Log(level, t, mt, caller, fullpath.String(), msg, allFields)
}

// LogFn implements [LoggerFn]
func (l *KLogger) LogFn(ctx context.Context, level Level, path string, skip int, fn FieldsFunc) {
	if level < l.minLevel {
		return
	}

	msg, fields := fn()
	l.Log(ctx, level, path, 1+skip, msg, fields)
}

// Sublogger implements [SubLogger] and creates a new sublogger
func (l *KLogger) Sublogger(path string, fields Fields) Logger {
	return &KLogger{
		minLevel:   l.minLevel,
		serializer: l.serializer,
		clock:      l.clock,
		path:       path,
		fields:     fields,
		parent:     l,
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

// LogFn computes fields conditionally to log
//
// If l implements [LoggerFn], then LogFn calls l.LogFn, else it will execute
// fn before calling l.Log.
func LogFn(l Logger, ctx context.Context, level Level, path string, skip int, fn FieldsFunc) {
	if fl, ok := l.(LoggerFn); ok {
		fl.LogFn(ctx, level, path, 1+skip, fn)
		return
	}
	msg, fields := fn()
	l.Log(ctx, level, path, 1+skip, msg, fields)
}

type (
	ctxKeyFields struct{}

	ctxFields struct {
		fields Fields
		parent *ctxFields
	}
)

func getCtxFields(ctx context.Context) *ctxFields {
	if ctx == nil {
		return nil
	}
	v := ctx.Value(ctxKeyFields{})
	if v == nil {
		return nil
	}
	return v.(*ctxFields)
}

func setCtxFields(ctx context.Context, fields *ctxFields) context.Context {
	return context.WithValue(ctx, ctxKeyFields{}, fields)
}

// WithFields adds log fields to context
func WithFields(ctx context.Context, fields Fields) context.Context {
	return ExtendCtx(ctx, ctx, fields)
}

// ExtendCtx adds log fields to context
func ExtendCtx(dest, ctx context.Context, fields Fields) context.Context {
	return setCtxFields(dest, &ctxFields{
		fields: fields,
		parent: getCtxFields(ctx),
	})
}
