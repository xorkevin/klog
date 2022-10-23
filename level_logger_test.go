package klog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"xorkevin.dev/kerrors"
)

func TestLevelLogger(t *testing.T) {
	t.Parallel()

	t.Run("logs at levels", func(t *testing.T) {
		t.Parallel()

		stackRegex := regexp.MustCompile(`Stack trace \[\S+ \S+:\d+\]`)

		assert := require.New(t)

		b := bytes.Buffer{}
		l := NewLevelLogger(New(OptMinLevel(LevelDebug), OptSerializer(NewJSONSerializer(NewSyncWriter(&b)))))
		l.Debug(context.Background(), "a debug msg", nil)
		l.DebugFn(context.Background(), func() (string, Fields) { return "another msg for debug", nil })
		l.Info(context.Background(), "an info msg", nil)
		l.InfoFn(context.Background(), func() (string, Fields) { return "some info msg", nil })
		l.Warn(context.Background(), "a warning", nil)
		l.WarnFn(context.Background(), func() (string, Fields) { return "warn msg", nil })
		l.Error(context.Background(), "error msg", nil)
		l.ErrorFn(context.Background(), func() (string, Fields) { return "error details", nil })
		l.Err(context.Background(), kerrors.WithMsg(nil, "something failed"), nil)
		l.Err(context.Background(), errors.New("plain error"), nil)
		l.WarnErr(context.Background(), kerrors.WithMsg(nil, "some warning"), nil)

		d := json.NewDecoder(&b)
		d.UseNumber()
		for _, i := range []struct {
			Level string
			Msg   string
		}{
			{
				Level: "DEBUG",
				Msg:   "a debug msg",
			},
			{
				Level: "DEBUG",
				Msg:   "another msg for debug",
			},
			{
				Level: "INFO",
				Msg:   "an info msg",
			},
			{
				Level: "INFO",
				Msg:   "some info msg",
			},
			{
				Level: "WARN",
				Msg:   "a warning",
			},
			{
				Level: "WARN",
				Msg:   "warn msg",
			},
			{
				Level: "ERROR",
				Msg:   "error msg",
			},
			{
				Level: "ERROR",
				Msg:   "error details",
			},
		} {
			var j map[string]interface{}
			assert.NoError(d.Decode(&j))
			assert.Equal(i.Level, j["level"])
			assert.Equal(i.Msg, j["msg"])
		}

		{
			var j map[string]interface{}
			assert.NoError(d.Decode(&j))
			assert.Equal("ERROR", j["level"])
			assert.Contains(j["caller"], "xorkevin.dev/klog/level_logger_test.go")
			assert.Contains(j["caller"], "xorkevin.dev/klog.TestLevelLogger")
			assert.Equal("something failed", j["msg"])
			errstr, ok := j["error"].(string)
			assert.True(ok)
			stackstr := stackRegex.FindString(errstr)
			assert.Contains(stackstr, "xorkevin.dev/klog/level_logger_test.go")
			assert.Contains(stackstr, "xorkevin.dev/klog.TestLevelLogger")
			assert.Equal("something failed: %!(STACKTRACE)", stackRegex.ReplaceAllString(errstr, "%!(STACKTRACE)"))
			stacktracestr, ok := j["stacktrace"].(string)
			assert.True(ok)
			assert.True(strings.HasPrefix(stacktracestr, "xorkevin.dev/klog.TestLevelLogger"))
		}
		{
			var j map[string]interface{}
			assert.NoError(d.Decode(&j))
			assert.Equal("ERROR", j["level"])
			assert.Equal("plain-error", j["msg"])
			assert.Equal("plain error", j["error"])
			assert.Equal("NONE", j["stacktrace"])
		}
		{
			var j map[string]interface{}
			assert.NoError(d.Decode(&j))
			assert.Equal("WARN", j["level"])
			assert.Equal("some warning", j["msg"])
			errstr, ok := j["error"].(string)
			assert.True(ok)
			stackstr := stackRegex.FindString(errstr)
			assert.Contains(stackstr, "xorkevin.dev/klog/level_logger_test.go")
			assert.Contains(stackstr, "xorkevin.dev/klog.TestLevelLogger")
			assert.Equal("some warning: %!(STACKTRACE)", stackRegex.ReplaceAllString(errstr, "%!(STACKTRACE)"))
			stacktracestr, ok := j["stacktrace"].(string)
			assert.True(ok)
			assert.True(strings.HasPrefix(stacktracestr, "xorkevin.dev/klog.TestLevelLogger"))
		}

		assert.False(d.More())
	})
}
