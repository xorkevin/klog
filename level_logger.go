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
	l.Logger.Log(ctx, LevelDebug, 1+l.Skip, msg, fields)
}

// DebugF logs at [LevelDebug]
func (l *LevelLogger) DebugF(ctx context.Context, fn FieldsFunc) {
	l.Logger.LogF(ctx, LevelDebug, 1+l.Skip, fn)
}

// Info logs at [LevelInfo]
func (l *LevelLogger) Info(ctx context.Context, msg string, fields Fields) {
	l.Logger.Log(ctx, LevelInfo, 1+l.Skip, msg, fields)
}

// InfoF logs at [LevelInfo]
func (l *LevelLogger) InfoF(ctx context.Context, fn FieldsFunc) {
	l.Logger.LogF(ctx, LevelInfo, 1+l.Skip, fn)
}

// Warn logs at [LevelWarn]
func (l *LevelLogger) Warn(ctx context.Context, msg string, fields Fields) {
	l.Logger.Log(ctx, LevelWarn, 1+l.Skip, msg, fields)
}

// WarnF logs at [LevelWarn]
func (l *LevelLogger) WarnF(ctx context.Context, fn FieldsFunc) {
	l.Logger.LogF(ctx, LevelWarn, 1+l.Skip, fn)
}

func getErrFields(err error) (string, Fields) {
	msg := "plain-error"
	if msger, ok := err.(kerrors.ErrorMsger); ok {
		msg = msger.ErrorMsg()
	}
	stacktrace := "NONE"
	m := kerrors.StackStringerMatcher{}
	if errors.As(err, &m) {
		stacktrace = m.StackStringer.StackString()
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
	l.Logger.Log(ctx, LevelWarn, 1+l.Skip, msg, allFields)
}

// Error logs at [LevelError]
func (l *LevelLogger) Error(ctx context.Context, msg string, fields Fields) {
	l.Logger.Log(ctx, LevelError, 1+l.Skip, msg, fields)
}

// ErrorF logs at [LevelError]
func (l *LevelLogger) ErrorF(ctx context.Context, fn FieldsFunc) {
	l.Logger.LogF(ctx, LevelError, 1+l.Skip, fn)
}

// Err logs an error [LevelError]
func (l *LevelLogger) Err(ctx context.Context, err error, fields Fields) {
	msg, allFields := getErrFields(err)
	mergeFields(allFields, fields)
	l.Logger.Log(ctx, LevelError, 1+l.Skip, msg, allFields)
}
