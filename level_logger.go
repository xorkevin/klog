package klog

import (
	"context"
	"errors"

	"golang.org/x/exp/slog"
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
func (l *LevelLogger) Debug(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.Logger.Log(ctx, slog.LevelDebug, 1+l.Skip, msg, attrs...)
}

// Info logs at [LevelInfo]
func (l *LevelLogger) Info(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.Logger.Log(ctx, slog.LevelInfo, 1+l.Skip, msg, attrs...)
}

// Warn logs at [LevelWarn]
func (l *LevelLogger) Warn(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.Logger.Log(ctx, slog.LevelWarn, 1+l.Skip, msg, attrs...)
}

func getErrAttr(err error) (string, slog.Attr) {
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
	return msg, slog.Group(
		"err",
		slog.String("msg", err.Error()),
		slog.String("trace", stacktrace),
	)
}

// WarnErr logs at [LevelWarn]
func (l *LevelLogger) WarnErr(ctx context.Context, err error, attrs ...slog.Attr) {
	msg, attr := getErrAttr(err)
	l.Logger.Log(ctx, slog.LevelWarn, 1+l.Skip, msg, append([]slog.Attr{attr}, attrs...)...)
}

// Error logs at [LevelError]
func (l *LevelLogger) Error(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.Logger.Log(ctx, slog.LevelError, 1+l.Skip, msg, attrs...)
}

// Err logs an error [LevelError]
func (l *LevelLogger) Err(ctx context.Context, err error, attrs ...slog.Attr) {
	msg, attr := getErrAttr(err)
	l.Logger.Log(ctx, slog.LevelError, 1+l.Skip, msg, append([]slog.Attr{attr}, attrs...)...)
}
