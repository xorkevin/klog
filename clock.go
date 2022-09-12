package klog

import "time"

type (
	RealTime struct {
	}
)

func (c RealTime) Time() time.Time {
	return time.Now().UTC().Round(0)
}
