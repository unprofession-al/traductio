package inputreader

import (
	"text/template"
	"time"

	"github.com/tj/go-naturaldate"
)

func getTemplateFuncMap() template.FuncMap {
	funcMap := template.FuncMap{
		"unixMilliTimestamp": unixMilliTimestamp,
		"unixTimestamp":      unixTimestamp,
	}
	return funcMap
}

func unixMilliTimestamp(in string) (int64, error) {
	t, err := timestamp(in)
	return t.UnixMilli(), err
}

func unixTimestamp(in string) (int64, error) {
	t, err := timestamp(in)
	return t.Unix(), err
}

func timestamp(in string) (time.Time, error) {
	out, err := time.Parse(time.RFC3339, in)
	if err == nil {
		return out, err
	}

	return naturaldate.Parse(in, time.Now())
}
