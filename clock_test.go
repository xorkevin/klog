package klog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRealTime(t *testing.T) {
	t.Parallel()

	t.Run("Time", func(t *testing.T) {
		t.Parallel()

		assert := require.New(t)

		var c Clock = RealTime{}
		assert.False(c.Time().IsZero())
	})
}
