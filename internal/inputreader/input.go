package inputreader

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"
	"text/template"
)

type InputConfig struct {
	URL              string            `json:"url" yaml:"url"`
	Headers          map[string]string `json:"headers" yaml:"headers"`
	Method           string            `json:"method" yaml:"method"`
	Body             string            `json:"body" yaml:"body"`
	HTTPExpectStatus int               `json:"http_expect_status" yaml:"http_expect_status"`
}

type Input struct {
	URL              string            `json:"url" yaml:"url"`
	Headers          map[string]string `json:"headers" yaml:"headers"`
	Method           string            `json:"method" yaml:"method"`
	Body             string            `json:"body" yaml:"body"`
	HTTPExpectStatus int               `json:"http_expect_status" yaml:"http_expect_status"`
}

func NewInput(c InputConfig, vars map[string]string) (Input, error) {
	// prepare input
	in := Input{
		Method:           c.Method,
		Headers:          map[string]string{},
		HTTPExpectStatus: c.HTTPExpectStatus,
	}

	// rendering func
	renderTemplate := func(t, hint string, v map[string]string) (string, error) {
		templ, err := template.New("template").Funcs(getTemplateFuncMap()).Parse(t)
		if err != nil {
			return "", fmt.Errorf("could not parse %s template: %s", hint, err.Error())
		}
		b := &bytes.Buffer{}
		err = templ.Execute(b, v)
		if err != nil {
			return "", fmt.Errorf("could not render %s template: %s", hint, err.Error())
		}
		return b.String(), nil
	}

	var err error

	// redering body
	in.Body, err = renderTemplate(c.Body, "body", vars)
	if err != nil {
		return in, err
	}

	// rendering URL
	in.URL, err = renderTemplate(c.URL, "body", vars)
	if err != nil {
		return in, err
	}

	// rendering headers
	for k, v := range c.Headers {
		name, err := renderTemplate(k, fmt.Sprintf("header name '%s'", k), vars)
		if err != nil {
			return in, err
		}

		in.Headers[name], err = renderTemplate(v, fmt.Sprintf("header value '%s'", k), vars)
		if err != nil {
			return in, err
		}
	}

	return in, nil
}

func (in Input) Fetch() ([]byte, error) {
	var err error
	var data []byte
	var status int

	u, err := url.Parse(in.URL)
	if err != nil {
		return []byte{}, err
	}

	if u.Scheme == "" {
		return readFile(u.Path)
	} else if u.Scheme == "http" || u.Scheme == "https" {
		data, status, err = readHypertext(in.URL, in.Body, in.Method, in.Headers)
		if err == nil && in.HTTPExpectStatus != 0 && status != in.HTTPExpectStatus {
			return data, fmt.Errorf("HTTP status code is %d, %d was expected.\n%s", status, in.HTTPExpectStatus, string(data))
		}
		return data, err
	} else if u.Scheme == "s3" {
		return readS3(u.Host, strings.TrimPrefix(u.Path, "/"))
	} else {
		return data, fmt.Errorf("cannot read %s: unsupported protocol %s", in.URL, u.Scheme)
	}
}
