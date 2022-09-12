package klog

import (
	"bytes"
	"errors"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type (
	failWriter struct {
	}
)

func (w failWriter) Write(p []byte) (int, error) {
	return 0, errors.New("Failed writing")
}

func TestJSONSerializer(t *testing.T) {
	t.Parallel()

	t.Run("handles log errors", func(t *testing.T) {
		t.Parallel()

		assert := require.New(t)

		s := NewJSONSerializer(failWriter{})
		b := bytes.Buffer{}
		s.ErrorLog = log.New(&b, "", log.LstdFlags|log.LUTC)
		s.Log(LevelInfo, time.Now().UTC().Round(0), nil, "", "hello", nil)

		assert.Contains(b.String(), "Failed writing")
	})
}
