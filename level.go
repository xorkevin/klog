package klog

type (
	// Level is a log level
	Level int
)

// Log levels
const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelNone
)

// String implements [fmt.Stringer]
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelNone:
		return "NONE"
	default:
		return "UNSET"
	}
}

// MarshalText implements [encoding.TextMarshaler]
func (l Level) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}

// UnmarshalText implements [encoding.TextUnmarshaler]
func (l *Level) UnmarshalText(data []byte) error {
	switch string(data) {
	case "DEBUG":
		*l = LevelDebug
	case "INFO":
		*l = LevelInfo
	case "WARN":
		*l = LevelWarn
	case "ERROR":
		*l = LevelError
	case "NONE":
		*l = LevelNone
	default:
		*l = LevelInfo
	}
	return nil
}
