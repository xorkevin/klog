package klog

import (
	"fmt"
	"runtime"
)

type (
	// callerFrame is a logger caller frame
	callerFrame struct {
		Function string
		File     string
		Line     int
	}
)

func linecaller(pc uintptr) callerFrame {
	frame, _ := runtime.CallersFrames([]uintptr{pc}).Next()
	return callerFrame{
		Function: frame.Function,
		File:     frame.File,
		Line:     frame.Line,
	}
}

func (f callerFrame) String() string {
	return fmt.Sprintf("%s %s:%d", f.Function, f.File, f.Line)
}
