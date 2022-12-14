package klog

import (
	"context"
	"errors"

	"xorkevin.dev/kerrors"
)

type (
	// LevelLogger provides convenience methods to log at particular levels
	LevelLogger struct {
		Logger Logger
		Skip   int
	}
)

// NewLevelLogger creates a new [*LevelLogger]
func NewLevelLogger(l Logger) *LevelLogger {
	return &LevelLogger{
		Logger: l,
		Skip:   1,
	}
}

// Debug logs at [LevelDebug]
func (l *LevelLogger) Debug(ctx context.Context, msg string, fields Fields) {
	l.Logger.Log(ctx, LevelDebug, "", 1+l.Skip, msg, fields)
}

// DebugFn logs at [LevelDebug]
func (l *LevelLogger) DebugFn(ctx context.Context, fn FieldsFunc) {
	LogFn(l.Logger, ctx, LevelDebug, "", 1+l.Skip, fn)
}

// Info logs at [LevelInfo]
func (l *LevelLogger) Info(ctx context.Context, msg string, fields Fields) {
	l.Logger.Log(ctx, LevelInfo, "", 1+l.Skip, msg, fields)
}

// InfoFn logs at [LevelInfo]
func (l *LevelLogger) InfoFn(ctx context.Context, fn FieldsFunc) {
	LogFn(l.Logger, ctx, LevelInfo, "", 1+l.Skip, fn)
}

// Warn logs at [LevelWarn]
func (l *LevelLogger) Warn(ctx context.Context, msg string, fields Fields) {
	l.Logger.Log(ctx, LevelWarn, "", 1+l.Skip, msg, fields)
}

// WarnFn logs at [LevelWarn]
func (l *LevelLogger) WarnFn(ctx context.Context, fn FieldsFunc) {
	LogFn(l.Logger, ctx, LevelWarn, "", 1+l.Skip, fn)
}

func getErrFields(err error) (string, Fields) {
	msg := "plain-error"
	var msger kerrors.ErrorMsger
	if errors.As(err, &msger) {
		msg = msger.ErrorMsg()
	}
	stacktrace := "NONE"
	var stackstringer kerrors.StackStringer
	if errors.As(err, &stackstringer) {
		stacktrace = stackstringer.StackString()
	}
	return msg, Fields{
		"error":      err.Error(),
		"stacktrace": stacktrace,
	}
}

// WarnErr logs at [LevelWarn]
func (l *LevelLogger) WarnErr(ctx context.Context, err error, fields Fields) {
	msg, allFields := getErrFields(err)
	mergeFields(allFields, fields)
	l.Logger.Log(ctx, LevelWarn, "", 1+l.Skip, msg, allFields)
}

// Error logs at [LevelError]
func (l *LevelLogger) Error(ctx context.Context, msg string, fields Fields) {
	l.Logger.Log(ctx, LevelError, "", 1+l.Skip, msg, fields)
}

// ErrorFn logs at [LevelError]
func (l *LevelLogger) ErrorFn(ctx context.Context, fn FieldsFunc) {
	LogFn(l.Logger, ctx, LevelError, "", 1+l.Skip, fn)
}

// Err logs an error [LevelError]
func (l *LevelLogger) Err(ctx context.Context, err error, fields Fields) {
	msg, allFields := getErrFields(err)
	mergeFields(allFields, fields)
	l.Logger.Log(ctx, LevelError, "", 1+l.Skip, msg, allFields)
}
