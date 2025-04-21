package klog

import (
	"context"
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

// WarnErr logs at [LevelWarn]
func (l *LevelLogger) WarnErr(ctx context.Context, err error, attrs ...Attr) {
	l.Logger.Log(ctx, LevelWarn, 1+l.Skip, err.Error(), AAny("err", err), AGroup("", attrs...))
}

// Error logs at [LevelError]
func (l *LevelLogger) Error(ctx context.Context, msg string, attrs ...Attr) {
	l.Logger.Log(ctx, LevelError, 1+l.Skip, msg, attrs...)
}

// Err logs an error [LevelError]
func (l *LevelLogger) Err(ctx context.Context, err error, attrs ...Attr) {
	l.Logger.Log(ctx, LevelError, 1+l.Skip, err.Error(), AAny("err", err), AGroup("", attrs...))
}
