package klog

import (
	"errors"
	"fmt"
	"math"
	"time"
)

type (
	// Kind is the kind of an attribute value
	Kind int

	// Value is an attribute value
	Value struct {
		kind Kind
		n    uint64
		s    string
		v    any
	}

	// LogValuer returns a Value for logging
	LogValuer interface {
		LogValue() Value
	}
)

// Attribute value kinds
const (
	KindAny Kind = iota
	KindBool
	KindInt64
	KindUint64
	KindFloat64
	KindString
	KindTime
	KindDuration
)

func ValueAny(v any) Value {
	switch v := v.(type) {
	case bool:
		return ValueBool(v)
	case int:
		return ValueInt64(int64(v))
	case int8:
		return ValueInt64(int64(v))
	case int16:
		return ValueInt64(int64(v))
	case int32:
		return ValueInt64(int64(v))
	case int64:
		return ValueInt64(v)
	case uint:
		return ValueUint64(uint64(v))
	case uint8:
		return ValueUint64(uint64(v))
	case uint16:
		return ValueUint64(uint64(v))
	case uint32:
		return ValueUint64(uint64(v))
	case uint64:
		return ValueUint64(v)
	case float32:
		return ValueFloat64(float64(v))
	case float64:
		return ValueFloat64(v)
	case string:
		return ValueString(v)
	case time.Time:
		return ValueTime(v)
	case time.Duration:
		return ValueDuration(v)
	case Value:
		return v
	default:
		return Value{
			kind: KindAny,
			v:    v,
		}
	}
}

func ValueGroup(v ...Attr) Value {
	return Value{
		kind: KindAny,
		v:    v,
	}
}

func ValueBool(v bool) Value {
	var n uint64 = 0
	if v {
		n = 1
	}
	return Value{
		kind: KindBool,
		n:    n,
	}
}

func ValueInt(v int) Value {
	return ValueInt64(int64(v))
}

func ValueInt64(v int64) Value {
	return Value{
		kind: KindInt64,
		n:    uint64(v),
	}
}

func ValueUint64(v uint64) Value {
	return Value{
		kind: KindUint64,
		n:    v,
	}
}

func ValueFloat64(v float64) Value {
	return Value{
		kind: KindFloat64,
		n:    math.Float64bits(v),
	}
}

func ValueString(v string) Value {
	return Value{
		kind: KindString,
		s:    v,
	}
}

func ValueTime(v time.Time) Value {
	return Value{
		kind: KindTime,
		v:    v,
	}
}

func ValueDuration(v time.Duration) Value {
	return Value{
		kind: KindDuration,
		n:    uint64(v),
	}
}

func (v Value) Kind() Kind {
	switch v.kind {
	case KindBool:
		return KindBool
	case KindInt64:
		return KindInt64
	case KindUint64:
		return KindUint64
	case KindFloat64:
		return KindFloat64
	case KindString:
		return KindString
	case KindTime:
		return KindTime
	case KindDuration:
		return KindDuration
	default:
		return KindAny
	}
}

func (v Value) Any() any {
	switch v.Kind() {
	case KindBool:
		return v.Bool()
	case KindInt64:
		return v.Int64()
	case KindUint64:
		return v.Uint64()
	case KindFloat64:
		return v.Float64()
	case KindString:
		return v.StringValue()
	case KindTime:
		return v.Time()
	case KindDuration:
		return v.Duration()
	default:
		return v.v
	}
}

func (v Value) Bool() bool {
	return v.n != 0
}

func (v Value) Int64() int64 {
	return int64(v.n)
}

func (v Value) Uint64() uint64 {
	return v.n
}

func (v Value) Float64() float64 {
	return math.Float64frombits(v.n)
}

func (v Value) StringValue() string {
	return v.s
}

func (v Value) Time() time.Time {
	t, ok := v.v.(time.Time)
	if !ok {
		return time.Time{}
	}
	return t
}

func (v Value) Duration() time.Duration {
	return time.Duration(v.n)
}

const (
	valueResolveRecursionLimit = 64
)

var (
	// ErrorExceedValueResolveRecursion is returned as a log value when Resolve
	// exceeds recursion limits.
	ErrorExceedValueResolveRecursion = errors.New("Exceeded value resolve recursion")
)

// Resolve recursively calls [LogValuer] LogValue up to a recursion limit. It
// resolves group value attributes recursively up to a recursion limit.
func (v Value) Resolve() Value {
	return v.resolveGroup(valueResolveRecursionLimit)
}

func (v Value) resolveGroup(depth int) Value {
	v = v.resolveValue()
	if v.kind != KindAny {
		return v
	}
	k, ok := v.v.([]Attr)
	if !ok {
		return v
	}
	for i, a := range k {
		if depth < 0 {
			k[i].Value = ValueAny(fmt.Errorf("%w: group value depth"))
		} else {
			k[i].Value = a.Value.resolveGroup(depth - 1)
		}
	}
	return v
}

func (v Value) resolveValue() Value {
	orig := v
	for i := 0; i < valueResolveRecursionLimit; i++ {
		if v.kind != KindAny {
			return v
		}
		k, ok := v.v.(LogValuer)
		if !ok {
			return v
		}
		v = k.LogValue()
	}
	return ValueAny(fmt.Errorf("%w: value type %T", orig.v))
}
