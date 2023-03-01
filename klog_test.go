package klog

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
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
			t.Log("caller", j)
			assert.True(ok)
			delete(j, "caller")
			assert.Equal(tc.Exp, j)
			assert.Contains(caller["fn"], "xorkevin.dev/klog.TestKLogger")
			assert.Contains(caller["src"], "xorkevin.dev/klog/klog_test.go")
			t.Log("fn", caller["fn"], caller["src"])
			assert.False(d.More())
		})
	}
}

//type (
//	minimalLogger struct {
//		clock      Clock
//		serializer Serializer
//	}
//)
//
//func (l *minimalLogger) Log(ctx context.Context, level Level, path string, skip int, msg string, fields Fields) {
//	t, mt := l.clock.Time()
//	caller := linecaller(1 + skip)
//	l.serializer.Log(level, t, mt, caller, path, msg, fields)
//}
//
//func TestSub(t *testing.T) {
//	t.Parallel()
//
//	ctx := WithFields(context.Background(), Fields{
//		"hello": "world",
//	})
//
//	for _, tc := range []struct {
//		Test  string
//		T     time.Time
//		MT    time.Time
//		Ctx   context.Context
//		Path  string
//		Fn    FieldsFunc
//		Exp   map[string]interface{}
//		Empty bool
//	}{
//		{
//			Test: "logs messages",
//			T:    time.Date(1991, time.August, 25, 20, 57, 8, 0, time.UTC),
//			MT:   time.Date(1991, time.August, 25, 20, 57, 9, 0, time.UTC),
//			Ctx: WithFields(ctx, Fields{
//				"f2": []string{"v2"},
//			}),
//			Path: "leaf",
//			Fn: func() (string, Fields) {
//				return "test message", Fields{
//					"f3":    "v3",
//					"hello": "foo",
//				}
//			},
//			Exp: map[string]interface{}{
//				"level":          "INFO",
//				"msg":            "test message",
//				"path":           "sublog.leaf",
//				"time":           "1991-08-25T20:57:08Z",
//				"unixtime":       json.Number(strconv.Itoa(683153828)),
//				"unixtimeus":     json.Number(strconv.Itoa(683153828000000)),
//				"monotime":       "1991-08-25T20:57:09Z",
//				"monounixtime":   json.Number(strconv.Itoa(683153829)),
//				"monounixtimeus": json.Number(strconv.Itoa(683153829000000)),
//				"f1":             "v11",
//				"hello":          "foo",
//				"f2":             []interface{}{"v2"},
//				"f3":             "v3",
//			},
//		},
//		{
//			Test: "handles nil context",
//			T:    time.Date(1991, time.August, 25, 20, 57, 8, 0, time.UTC),
//			MT:   time.Date(1991, time.August, 25, 20, 57, 9, 0, time.UTC),
//			Ctx:  nil,
//			Fn: func() (string, Fields) {
//				return "some message", nil
//			},
//			Exp: map[string]interface{}{
//				"level":          "INFO",
//				"msg":            "some message",
//				"path":           "sublog",
//				"time":           "1991-08-25T20:57:08Z",
//				"unixtime":       json.Number(strconv.Itoa(683153828)),
//				"unixtimeus":     json.Number(strconv.Itoa(683153828000000)),
//				"monotime":       "1991-08-25T20:57:09Z",
//				"monounixtime":   json.Number(strconv.Itoa(683153829)),
//				"monounixtimeus": json.Number(strconv.Itoa(683153829000000)),
//				"f1":             "v11",
//			},
//		},
//	} {
//		tc := tc
//		t.Run(tc.Test, func(t *testing.T) {
//			t.Parallel()
//
//			assert := require.New(t)
//
//			b := bytes.Buffer{}
//			var l Logger = &minimalLogger{
//				clock:      testClock{t: tc.T, mt: tc.MT},
//				serializer: NewJSONSerializer(NewSyncWriter(&b)),
//			}
//			l = Sub(l, "sublog", Fields{
//				"f1": "v11",
//			})
//			LogFn(l, tc.Ctx, LevelInfo, tc.Path, 1, tc.Fn)
//			{
//				msg, fields := tc.Fn()
//				l.Log(tc.Ctx, LevelInfo, tc.Path, 1, msg, fields)
//			}
//
//			if tc.Empty {
//				assert.Len(b.Bytes(), 0)
//				return
//			}
//			d := json.NewDecoder(&b)
//			d.UseNumber()
//			for i := 0; i < 2; i++ {
//				var j map[string]interface{}
//				assert.NoError(d.Decode(&j))
//				callerstr := j["caller"]
//				delete(j, "caller")
//				assert.Equal(tc.Exp, j)
//				assert.Contains(callerstr, "xorkevin.dev/klog.TestSub")
//				assert.Contains(callerstr, "xorkevin.dev/klog/klog_test.go")
//			}
//			assert.False(d.More())
//		})
//	}
//}
