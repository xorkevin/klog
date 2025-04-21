package klog

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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

	ctx := CtxWithAttrs(context.Background(), AString("hello", "world"))

	for _, tc := range []struct {
		Test  string
		Opts  []LoggerOpt
		T     time.Time
		Ctx   context.Context
		Msg   string
		Attrs []Attr
		Exp   map[string]any
		Empty bool
	}{
		{
			Test:  "logs messages",
			Opts:  []LoggerOpt{OptMinLevelStr("INFO"), OptSubhandler("base", AString("f1", "v1"))},
			T:     time.Date(1991, time.August, 25, 20, 57, 8, 0, time.UTC),
			Ctx:   CtxWithAttrs(ctx, AAny("f2", []string{"v2"})),
			Msg:   "test message",
			Attrs: []Attr{AString("f3", "v3"), AString("hello", "foo")},
			Exp: map[string]any{
				"level": "INFO",
				"msg":   "test message",
				"mod":   ".base.sublog",
				"t":     "1991-08-25T20:57:08Z",
				"f1":    "v1",
				"hello": "world",
				"f2":    []any{"v2"},
				"f3":    "v3",
			},
		},
		{
			Test: "handles nil context",
			Opts: []LoggerOpt{OptMinLevelStr("INFO"), OptSubhandler("", AString("f1", "v2"))},
			T:    time.Date(1991, time.August, 25, 20, 57, 8, 0, time.UTC),
			Ctx:  nil,
			Msg:  "some message",
			Exp: map[string]any{
				"level": "INFO",
				"msg":   "some message",
				"mod":   ".sublog",
				"t":     "1991-08-25T20:57:08Z",
				"f1":    "v2",
			},
		},
		{
			Test:  "below level",
			Opts:  []LoggerOpt{OptMinLevelStr("WARN"), OptSubhandler("", AString("f1", "v1"))},
			T:     time.Date(1991, time.August, 25, 20, 57, 8, 0, time.UTC),
			Ctx:   nil,
			Msg:   "some message",
			Empty: true,
		},
	} {
		t.Run(tc.Test, func(t *testing.T) {
			t.Parallel()

			assert := require.New(t)

			var b bytes.Buffer
			var l Logger = New(append([]LoggerOpt{OptHandler(NewJSONSlogHandler(NewSyncWriter(&b))), OptClock(testClock{t: tc.T})}, tc.Opts...)...)
			l = l.Sublogger("sublog", AString("f1", "v11"))
			l.Log(tc.Ctx, LevelInfo, 0, tc.Msg, tc.Attrs...)

			if tc.Empty {
				assert.Equal(0, b.Len())
				return
			}
			d := json.NewDecoder(&b)
			d.UseNumber()
			var j map[string]any
			assert.NoError(d.Decode(&j))
			src, ok := j["src"].(map[string]any)
			assert.True(ok)
			delete(j, "src")
			assert.Equal(tc.Exp, j)
			srcfn, ok := src["fn"].(string)
			assert.True(ok)
			assert.True(strings.HasPrefix(srcfn, "xorkevin.dev/klog.TestKLogger"))
			srcfile, ok := src["file"].(string)
			assert.True(ok)
			assert.True(strings.HasPrefix(srcfile, "xorkevin.dev/klog/klog_test.go"))
			assert.False(d.More())
		})
	}
}

func TestDiscard(t *testing.T) {
	t.Parallel()

	var _ Logger = Discard
	var _ Handler = DiscardHandler
}
