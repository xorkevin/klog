package klog

import (
	"io"
	"sync"
)

type (
	// SyncWriter is a thread safe writer
	SyncWriter struct {
		mu *sync.Mutex
		w  io.Writer
	}
)

// NewSyncWriter creates a new [*SyncWriter]
func NewSyncWriter(w io.Writer) *SyncWriter {
	return &SyncWriter{
		mu: &sync.Mutex{},
		w:  w,
	}
}

// Write implements [io.Writer]
func (w *SyncWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.w.Write(p)
}
