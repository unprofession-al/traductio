package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
	"traductio/internal/sink"

	"github.com/itchyny/gojq"
)

func Process(j []byte, i Iterator, inherited sink.Point, test bool) ([]sink.Point, bool, string, error) {
	results := []sink.Point{}

	selected, err := queryList(j, i.Selector)
	if err != nil {
		return results, false, "", err
	}

	var elements []interface{}

	err = json.Unmarshal(selected, &elements)
	if err != nil {
		return results, false, "", err
	}

	for _, element := range elements {
		point := inherited.Copy()
		elem, err := json.MarshalIndent(element, "", "  ")

		if err != nil {
			return results, false, "", err
		}

		if i.Time.Selector != "" {
			if i.Time.Format == "unixMilliTimestamp" {
				out, err := queryValue(elem, i.Time.Selector)
				if err != nil {
					return results, false, "", err
				}
				point.Timestamp = time.Unix(int64(out)/1000, 0)
			} else if i.Time.Format == "unixTimestamp" {
				out, err := queryValue(elem, i.Time.Selector)
				if err != nil {
					return results, false, "", err
				}
				point.Timestamp = time.Unix(int64(out), 0)
			} else {
				out, err := queryBytes(elem, i.Time.Selector)
				if err != nil {
					return results, false, "", err
				}
				point.Timestamp, err = time.Parse(i.Time.Format, string(out))
				if err != nil {
					return results, false, "", err
				}
			}
		}

		for key, selector := range i.Values {
			out, err := queryValue(elem, selector)
			if err != nil {
				return results, false, "", err
			}
			point.Values[key] = out
		}

		for key, value := range i.FixedValues {
			point.Values[key] = value
		}

		for key, selector := range i.Tags {
			out, err := queryBytes(elem, selector)
			if err != nil {
				return results, false, "", err
			}
			trimmed := strings.Trim(string(out), "\"\\")
			point.Tags[key] = trimmed
		}

		for key, value := range i.FixedTags {
			point.Tags[key] = value
		}

		if i.Iterator != nil {
			processed, stop, jsonFragment, err := Process(elem, *i.Iterator, point, test)
			if err != nil {
				return results, false, "", err
			}
			results = append(results, processed...)
			if stop {
				return results, stop, jsonFragment, nil
			}
		} else {
			if !point.IsEmpty() {
				results = append(results, point)
			}
			if test {
				return results, true, string(elem), nil
			}
		}
	}

	return results, false, "", nil
}

func queryList(j []byte, q string) ([]byte, error) {
	//j = bytes.ReplaceAll(j, []byte("buckets\":null"), []byte("buckets\":[]"))
	var input map[string]interface{}
	err := json.Unmarshal(j, &input)
	if err != nil {
		return nil, err
	}

	query, err := gojq.Parse(q)
	if err != nil {
		log.Fatalln(err)
	}

	var out []interface{}
	iter := query.Run(input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return []byte{}, err
		}
		out = append(out, v)
	}
	return json.Marshal(out)
}

func queryBytes(j []byte, q string) ([]byte, error) {
	//j = bytes.ReplaceAll(j, []byte("buckets\":null"), []byte("buckets\":[]"))
	var input map[string]interface{}
	err := json.Unmarshal(j, &input)
	if err != nil {
		return nil, err
	}

	query, err := gojq.Parse(q)
	if err != nil {
		log.Fatalln(err)
	}

	var out interface{}
	iter := query.Run(input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return []byte{}, err
		}
		out = v
	}
	return json.Marshal(out)
}

func queryValue(j []byte, q string) (float64, error) {
	j = bytes.ReplaceAll(j, []byte("buckets\":null"), []byte("buckets\":[]"))

	var input map[string]interface{}
	err := json.Unmarshal(j, &input)
	if err != nil {
		return 0.0, err
	}

	query, err := gojq.Parse(q)
	if err != nil {
		log.Fatalln(err)
	}

	var out float64
	iter := query.Run(input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return out, err
		}
		if out, ok = v.(float64); !ok {
			return out, fmt.Errorf("could not read value as float64")
		}

	}
	return out, nil
}
