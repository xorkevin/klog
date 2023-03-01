package klog

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
)

type (
	testClock struct {
		t time.Time
	}
)

func (c testClock) Time() time.Time {
	return c.t
}

func TestKLogger(t *testing.T) {
	t.Parallel()

	ctx := CtxWithAttrs(context.Background(), slog.String("hello", "world"))

	for _, tc := range []struct {
		Test  string
		Opts  []LoggerOpt
		T     time.Time
		Ctx   context.Context
		Msg   string
		Attrs []slog.Attr
		Exp   map[string]interface{}
		Empty bool
	}{
		{
			Test:  "logs messages",
			Opts:  []LoggerOpt{OptMinLevelStr("INFO"), OptSubhandler("base", []slog.Attr{slog.String("f1", "v1")})},
			T:     time.Date(1991, time.August, 25, 20, 57, 8, 0, time.UTC),
			Ctx:   CtxWithAttrs(ctx, slog.Any("f2", []string{"v2"})),
			Msg:   "test message",
			Attrs: []slog.Attr{slog.String("f3", "v3"), slog.String("hello", "foo")},
			Exp: map[string]interface{}{
				"level": "INFO",
				"msg":   "test message",
				"path":  ".base.sublog",
				"t": map[string]interface{}{
					"time":    "1991-08-25T20:57:08Z",
					"unix_us": json.Number(strconv.Itoa(683153828000000)),
					"mono_us": json.Number(strconv.Itoa(683153828000000)),
				},
				"f1":    "v1",
				"hello": "foo",
				"f2":    []interface{}{"v2"},
				"f3":    "v3",
			},
		},
		{
			Test: "handles nil context",
			Opts: []LoggerOpt{OptMinLevelStr("INFO"), OptSubhandler("", []slog.Attr{slog.String("f1", "v2")})},
			T:    time.Date(1991, time.August, 25, 20, 57, 8, 0, time.UTC),
			Ctx:  nil,
			Msg:  "some message",
			Exp: map[string]interface{}{
				"level": "INFO",
				"msg":   "some message",
				"path":  ".sublog",
				"t": map[string]interface{}{
					"time":    "1991-08-25T20:57:08Z",
					"unix_us": json.Number(strconv.Itoa(683153828000000)),
					"mono_us": json.Number(strconv.Itoa(683153828000000)),
				},
				"f1": "v2",
			},
		},
		{
			Test:  "below level",
			Opts:  []LoggerOpt{OptMinLevelStr("WARN"), OptSubhandler("", []slog.Attr{slog.String("f1", "v1")})},
			T:     time.Date(1991, time.August, 25, 20, 57, 8, 0, time.UTC),
			Ctx:   nil,
			Msg:   "some message",
			Empty: true,
		},
	} {
		tc := tc
		t.Run(tc.Test, func(t *testing.T) {
			t.Parallel()

			assert := require.New(t)

			var b bytes.Buffer
			var l Logger = New(append([]LoggerOpt{OptHandler(NewJSONSlogHandler(NewSyncWriter(&b))), OptClock(testClock{t: tc.T})}, tc.Opts...)...)
			l = l.Sublogger("sublog", []slog.Attr{slog.String("f1", "v11")})
			l.Log(tc.Ctx, slog.LevelInfo, 0, tc.Msg, tc.Attrs...)

			if tc.Empty {
				assert.Equal(0, b.Len())
				return
			}
			d := json.NewDecoder(&b)
			d.UseNumber()
			var j map[string]interface{}
			assert.NoError(d.Decode(&j))
			caller, ok := j["caller"].(map[string]interface{})
			assert.True(ok)
			delete(j, "caller")
			assert.Equal(tc.Exp, j)
			callerfn, ok := caller["fn"].(string)
			assert.True(ok)
			assert.True(strings.HasPrefix(callerfn, "xorkevin.dev/klog.TestKLogger"))
			callersrc, ok := caller["src"].(string)
			assert.True(ok)
			assert.True(strings.HasPrefix(callersrc, "xorkevin.dev/klog/klog_test.go"))
			assert.False(d.More())
		})
	}
}
