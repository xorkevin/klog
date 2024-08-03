package klog

import "time"

type (
	// RealTime is a real time clock
	RealTime struct{}
)

func (c RealTime) Time() time.Time {
	return time.Now()
}
