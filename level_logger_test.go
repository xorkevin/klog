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
	"golang.org/x/exp/slog"
	"xorkevin.dev/kerrors"
)

func TestLevelLogger(t *testing.T) {
	t.Parallel()

	t.Run("logs at levels", func(t *testing.T) {
		t.Parallel()

		stackRegex := regexp.MustCompile(`Stack trace\n\[\[\n\S+ \S+:\d+\n\]\]`)

		assert := require.New(t)

		var b bytes.Buffer
		l := NewLevelLogger(New(OptMinLevel(slog.LevelDebug), OptHandler(NewJSONSlogHandler(NewSyncWriter(&b)))))
		l.Debug(context.Background(), "a debug msg")
		l.Info(context.Background(), "an info msg")
		l.Warn(context.Background(), "a warning")
		l.Error(context.Background(), "error msg")
		l.Err(context.Background(), kerrors.WithMsg(nil, "something failed"))
		l.Err(context.Background(), errors.New("plain error"))
		l.WarnErr(context.Background(), kerrors.WithMsg(nil, "some warning"))

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
				Level: "INFO",
				Msg:   "an info msg",
			},
			{
				Level: "WARN",
				Msg:   "a warning",
			},
			{
				Level: "ERROR",
				Msg:   "error msg",
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
			caller, ok := j["caller"].(map[string]interface{})
			assert.True(ok)
			callerfn, ok := caller["fn"].(string)
			assert.True(ok)
			assert.True(strings.HasPrefix(callerfn, "xorkevin.dev/klog.TestLevelLogger"))
			callersrc, ok := caller["src"].(string)
			assert.True(ok)
			assert.True(strings.HasPrefix(callersrc, "xorkevin.dev/klog/level_logger_test.go"))
			assert.Equal("something failed", j["msg"])
			logerr, ok := j["err"].(map[string]interface{})
			assert.True(ok)
			errmsg, ok := logerr["msg"].(string)
			assert.True(ok)
			stackstr := stackRegex.FindString(errmsg)
			assert.Contains(stackstr, "xorkevin.dev/klog/level_logger_test.go")
			assert.Contains(stackstr, "xorkevin.dev/klog.TestLevelLogger")
			assert.Equal("something failed\n--\n%!(STACKTRACE)", stackRegex.ReplaceAllString(errmsg, "%!(STACKTRACE)"))
			stacktracestr, ok := logerr["trace"].(string)
			assert.True(ok)
			assert.True(strings.HasPrefix(stacktracestr, "xorkevin.dev/klog.TestLevelLogger"))
		}
		{
			var j map[string]interface{}
			assert.NoError(d.Decode(&j))
			assert.Equal("ERROR", j["level"])
			assert.Equal("plain-error", j["msg"])
			logerr, ok := j["err"].(map[string]interface{})
			assert.True(ok)
			assert.Equal("plain error", logerr["msg"])
			assert.Equal("NONE", logerr["trace"])
		}
		{
			var j map[string]interface{}
			assert.NoError(d.Decode(&j))
			assert.Equal("WARN", j["level"])
			assert.Equal("some warning", j["msg"])
			logerr, ok := j["err"].(map[string]interface{})
			assert.True(ok)
			errstr, ok := logerr["msg"].(string)
			assert.True(ok)
			stackstr := stackRegex.FindString(errstr)
			assert.Contains(stackstr, "xorkevin.dev/klog/level_logger_test.go")
			assert.Contains(stackstr, "xorkevin.dev/klog.TestLevelLogger")
			assert.Equal("some warning\n--\n%!(STACKTRACE)", stackRegex.ReplaceAllString(errstr, "%!(STACKTRACE)"))
			stacktracestr, ok := logerr["trace"].(string)
			assert.True(ok)
			assert.True(strings.HasPrefix(stacktracestr, "xorkevin.dev/klog.TestLevelLogger"))
		}

		assert.False(d.More())
	})
}
