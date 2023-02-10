package klog

import (
	"time"

	"golang.org/x/exp/slog"
)

type (
	// Value is an attribute value
	Value struct {
		svalue slog.Value
	}
)

func ValueAny(v any) Value {
	return Value{
		svalue: slog.AnyValue(v),
	}
}

func ValueString(v string) Value {
	return Value{
		svalue: slog.StringValue(v),
	}
}

func ValueInt(v int) Value {
	return Value{
		svalue: slog.IntValue(v),
	}
}

func ValueInt64(v int64) Value {
	return Value{
		svalue: slog.Int64Value(v),
	}
}

func ValueUint64(v uint64) Value {
	return Value{
		svalue: slog.Uint64Value(v),
	}
}

func ValueFloat64(v float64) Value {
	return Value{
		svalue: slog.Float64Value(v),
	}
}

func ValueBool(v bool) Value {
	return Value{
		svalue: slog.BoolValue(v),
	}
}

func ValueGroup(v ...Attr) Value {
	return ValueAny(v)
}

func ValueTime(v time.Time) Value {
	return Value{
		svalue: slog.TimeValue(v),
	}
}

func ValueDuration(v time.Duration) Value {
	return Value{
		svalue: slog.DurationValue(v),
	}
}
