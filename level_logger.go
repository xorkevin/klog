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
		Skip:   0,
	}
}

// Debug logs at [LevelDebug]
func (l *LevelLogger) Debug(ctx context.Context, msg string, attrs ...Attr) {
	l.Logger.Log(ctx, LevelDebug, 1+l.Skip, msg, attrs...)
}

// Info logs at [LevelInfo]
func (l *LevelLogger) Info(ctx context.Context, msg string, attrs ...Attr) {
	l.Logger.Log(ctx, LevelInfo, 1+l.Skip, msg, attrs...)
}

// Warn logs at [LevelWarn]
func (l *LevelLogger) Warn(ctx context.Context, msg string, attrs ...Attr) {
	l.Logger.Log(ctx, LevelWarn, 1+l.Skip, msg, attrs...)
}

func getErrAttr(err error) (string, Attr) {
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
	return msg, AGroup(
		"err",
		AString("msg", err.Error()),
		AString("trace", stacktrace),
	)
}

// WarnErr logs at [LevelWarn]
func (l *LevelLogger) WarnErr(ctx context.Context, err error, attrs ...Attr) {
	msg, attr := getErrAttr(err)
	l.Logger.Log(ctx, LevelWarn, 1+l.Skip, msg, append([]Attr{attr}, attrs...)...)
}

// Error logs at [LevelError]
func (l *LevelLogger) Error(ctx context.Context, msg string, attrs ...Attr) {
	l.Logger.Log(ctx, LevelError, 1+l.Skip, msg, attrs...)
}

// Err logs an error [LevelError]
func (l *LevelLogger) Err(ctx context.Context, err error, attrs ...Attr) {
	msg, attr := getErrAttr(err)
	l.Logger.Log(ctx, LevelError, 1+l.Skip, msg, append([]Attr{attr}, attrs...)...)
}
