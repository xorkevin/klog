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
	}
}

// Log implements [Serializer]
func (w *JSONSerializer) Log(level Level, t time.Time, caller *Frame, path string, msg string, fields Fields) {
	timestr := t.Format(time.RFC3339Nano)
	unixtime := t.Unix()
	unixtimeus := t.UnixMicro()
	callerstr := ""
	if caller != nil {
		callerstr = caller.String()
	}
	allFields := Fields{
		w.FieldLevel:      level.String(),
		w.FieldTime:       timestr,
		w.FieldTimeUnix:   unixtime,
		w.FieldTimeUnixUS: unixtimeus,
		w.FieldCaller:     callerstr,
		w.FieldPath:       path,
		w.FieldMsg:        msg,
	}
	mergeFields(allFields, fields)
	b := bytes.Buffer{}
	j := json.NewEncoder(&b)
	j.SetEscapeHTML(false)
	if err := j.Encode(allFields); err != nil {
		log.Println(err)
		return
	}
	if _, err := io.Copy(w.W, &b); err != nil {
		log.Println(err)
		return
	}
}
