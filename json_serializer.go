package klog

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"time"
)

type (
	// JSONSerializer writes logs in json format
	JSONSerializer struct {
		FieldLevel      string
		FieldTime       string
		FieldTimeUnix   string
		FieldTimeUnixUS string
		FieldCaller     string
		FieldPath       string
		FieldMsg        string
		W               io.Writer
		ErrorLog        *log.Logger
	}
)

// NewJSONSerializer creates a new [*JSONSerializer]
func NewJSONSerializer(w io.Writer) *JSONSerializer {
	return &JSONSerializer{
		FieldLevel:      "level",
		FieldTime:       "time",
		FieldTimeUnix:   "unixtime",
		FieldTimeUnixUS: "unixtimeus",
		FieldCaller:     "caller",
		FieldPath:       "path",
		FieldMsg:        "msg",
		W:               w,
		ErrorLog:        log.New(io.Discard, "", log.LstdFlags),
	}
}

// Log implements [Serializer]
func (s *JSONSerializer) Log(level Level, t time.Time, caller *Frame, path string, msg string, fields Fields) {
	timestr := t.Format(time.RFC3339Nano)
	unixtime := t.Unix()
	unixtimeus := t.UnixMicro()
	callerstr := ""
	if caller != nil {
		callerstr = caller.String()
	}
	allFields := Fields{
		s.FieldLevel:      level.String(),
		s.FieldTime:       timestr,
		s.FieldTimeUnix:   unixtime,
		s.FieldTimeUnixUS: unixtimeus,
		s.FieldCaller:     callerstr,
		s.FieldPath:       path,
		s.FieldMsg:        msg,
	}
	mergeFields(allFields, fields)
	b := bytes.Buffer{}
	j := json.NewEncoder(&b)
	j.SetEscapeHTML(false)
	if err := j.Encode(allFields); err != nil {
		s.ErrorLog.Println(err)
		return
	}
	if _, err := io.Copy(s.W, &b); err != nil {
		s.ErrorLog.Println(err)
		return
	}
}
