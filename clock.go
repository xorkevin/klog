package klog

import "time"

type (
	// RealTime is a real time clock
	RealTime struct {
	}
)

func (c RealTime) Time() (time.Time, time.Time) {
	t := time.Now().UTC()
	return t, t.Round(0)
}
