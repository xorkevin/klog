package klog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLevel(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		Test  string
		Level Level
	}{
		{
			Test:  "DEBUG",
			Level: LevelDebug,
		},
		{
			Test:  "INFO",
			Level: LevelInfo,
		},
		{
			Test:  "WARN",
			Level: LevelWarn,
		},
		{
			Test:  "ERROR",
			Level: LevelError,
		},
		{
			Test:  "NONE",
			Level: LevelNone,
		},
	} {
		tc := tc
		t.Run(tc.Test, func(t *testing.T) {
			t.Parallel()

			assert := require.New(t)

			var level Level
			assert.NoError(level.UnmarshalText([]byte(tc.Test)))
			assert.Equal(tc.Level, level)
			assert.Equal(tc.Test, level.String())
		})
	}

	t.Run("BOGUS", func(t *testing.T) {
		t.Parallel()

		assert := require.New(t)

		var level Level
		assert.NoError(level.UnmarshalText([]byte("BOGUS")))
		assert.Equal(LevelInfo, level)
	})
	t.Run("UNSET", func(t *testing.T) {
		t.Parallel()

		assert := require.New(t)

		assert.Equal("UNSET", Level(-1).String())
	})
}
