package klog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"xorkevin.dev/kerrors"
)

func searchMapKey(data any, key string) map[string]any {
	switch v := data.(type) {
	case map[string]any:
		if _, ok := v[key]; ok {
			return v
		}
		for _, val := range v {
			if result := searchMapKey(val, key); result != nil {
				return result
			}
		}
	case []any:
		for _, val := range v {
			if result := searchMapKey(val, key); result != nil {
				return result
			}
		}
	}
	return nil
}

func TestLevelLogger(t *testing.T) {
	t.Parallel()

	t.Run("logs at levels", func(t *testing.T) {
		t.Parallel()

		assert := require.New(t)

		var b bytes.Buffer
		l := NewLevelLogger(New(OptMinLevel(LevelDebug), OptHandler(NewJSONSlogHandler(NewSyncWriter(&b)))))
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
			var j map[string]any
			assert.NoError(d.Decode(&j))
			assert.Equal(i.Level, j["level"])
			assert.Equal(i.Msg, j["msg"])
		}

		{
			var j map[string]any
			assert.NoError(d.Decode(&j))
			assert.Equal("ERROR", j["level"])
			src, ok := j["src"].(map[string]any)
			assert.True(ok)
			srcfn, ok := src["fn"].(string)
			assert.True(ok)
			assert.True(strings.HasPrefix(srcfn, "xorkevin.dev/klog.TestLevelLogger"))
			srcfile, ok := src["file"].(string)
			assert.True(ok)
			assert.True(strings.HasPrefix(srcfile, "xorkevin.dev/klog/level_logger_test.go"))
			assert.Equal("something failed", j["msg"])
			logerr, ok := j["err"].(map[string]any)
			assert.True(ok)
			stackTrace := searchMapKey(logerr, "stack")
			assert.NotNil(stackTrace)
			stack, ok := stackTrace["stack"].([]any)
			assert.True(ok)
			assert.NotNil(stack)
			assert.Contains(stack[0].(map[string]any)["file"], "xorkevin.dev/klog/level_logger_test.go")
			assert.Contains(stack[0].(map[string]any)["fn"], "xorkevin.dev/klog.TestLevelLogger")
			delete(stackTrace, "stack")
			assert.Equal(map[string]any{
				"msg": "something failed",
				"cause": map[string]any{
					"msg": "Stack trace",
				},
			}, logerr)
		}
		{
			var j map[string]any
			assert.NoError(d.Decode(&j))
			assert.Equal("ERROR", j["level"])
			assert.Equal("plain error", j["msg"])
			logerr, ok := j["err"].(string)
			assert.True(ok)
			assert.Equal("plain error", logerr)
		}
		{
			var j map[string]any
			assert.NoError(d.Decode(&j))
			assert.Equal("WARN", j["level"])
			assert.Equal("some warning", j["msg"])
			logerr, ok := j["err"].(map[string]any)
			assert.True(ok)
			stackTrace := searchMapKey(logerr, "stack")
			assert.NotNil(stackTrace)
			stack, ok := stackTrace["stack"].([]any)
			assert.True(ok)
			assert.NotNil(stack)
			assert.Contains(stack[0].(map[string]any)["file"], "xorkevin.dev/klog/level_logger_test.go")
			assert.Contains(stack[0].(map[string]any)["fn"], "xorkevin.dev/klog.TestLevelLogger")
			delete(stackTrace, "stack")
			assert.Equal(map[string]any{
				"msg": "some warning",
				"cause": map[string]any{
					"msg": "Stack trace",
				},
			}, logerr)
		}

		assert.False(d.More())
	})
}
