package main

import (
	"reflect"
	"testing"
	"time"
	"traductio/internal/sink"
)

var refTime = time.Unix(1642892400000/1000, 0)
var jsn = []byte(`{
	"by_time": [
		{
			"time": 1642892400000,
			"groups": [
				{
					"name": "foo",
					"values": {
						"count": 2,
						"volume": 4
					}
				},
				{
					"name": "bar",
					"values": {
						"count": 3,
						"volume": 5
					}
				}
			]
		}
	]
}`)

var processTestSets = []struct {
	name        string
	i           Iterator
	errExpected bool
	points      []sink.Point
}{
	{
		name:        "no_iterator",
		i:           Iterator{},
		errExpected: false,
		points:      []sink.Point{},
	},
	{
		name: "faulty_iterator",
		i: Iterator{
			Selector: ".by_time.groups[]",
		},
		errExpected: true,
		points:      []sink.Point{},
	},
	{
		name: "proper_iterator_no_time_no_tags_no_values",
		i: Iterator{
			Selector: ".by_time[]",
		},
		errExpected: false,
		points:      []sink.Point{},
	},
	{
		name: "proper_iterator_with_time_no_tags_no_values",
		i: Iterator{
			Selector: ".by_time[]",
			Time: TimeSet{
				Selector: ".time",
				Format:   "unixMilliTimestamp",
			},
		},
		errExpected: false,
		points:      []sink.Point{},
	},
	{
		name: "proper_iterator_with_time_with_tags_no_values",
		i: Iterator{
			Selector: ".by_time[]",
			Time: TimeSet{
				Selector: ".time",
				Format:   "unixMilliTimestamp",
			},
			Iterator: &Iterator{
				Selector: ".groups[]",
				Tags: map[string]string{
					"name": ".name",
				},
			},
		},
		errExpected: false,
		points:      []sink.Point{},
	},
	{
		name: "proper_iterator_with_time_with_tags_with_values",
		i: Iterator{
			Selector: ".by_time[]",
			Time: TimeSet{
				Selector: ".time",
				Format:   "unixMilliTimestamp",
			},
			Iterator: &Iterator{
				Selector: ".groups[]",
				Tags: map[string]string{
					"name": ".name",
				},
				Values: map[string]string{
					"count":  ".values.count",
					"volume": ".values.volume",
				},
			},
		},
		errExpected: false,
		points: []sink.Point{
			{
				Timestamp: refTime,
				Tags: map[string]string{
					"name": "foo",
				},
				Values: map[string]float64{
					"count":  2,
					"volume": 4,
				},
			},
			{
				Timestamp: refTime,
				Tags: map[string]string{
					"name": "bar",
				},
				Values: map[string]float64{
					"count":  3,
					"volume": 5,
				},
			},
		},
	},
	{
		name: "proper_iterator_with_time_no_tags_with_values",
		i: Iterator{
			Selector: ".by_time[]",
			Time: TimeSet{
				Selector: ".time",
				Format:   "unixMilliTimestamp",
			},
			Iterator: &Iterator{
				Selector: ".groups[]",
				Values: map[string]string{
					"count":  ".values.count",
					"volume": ".values.volume",
				},
			},
		},
		points: []sink.Point{
			{
				Timestamp: refTime,
				Values: map[string]float64{
					"count":  2.0,
					"volume": 4.0,
				},
				Tags: map[string]string{},
			},
			{
				Timestamp: refTime,
				Values: map[string]float64{
					"count":  3.0,
					"volume": 5.0,
				},
				Tags: map[string]string{},
			},
		},
		errExpected: false,
	},
	{
		name: "proper_iterator_no_time_no_tags_with_values",
		i: Iterator{
			Selector: ".by_time[]",
			Iterator: &Iterator{
				Selector: ".groups[]",
				Values: map[string]string{
					"count":  ".values.count",
					"volume": ".values.volume",
				},
			},
		},
		errExpected: false,
		points:      []sink.Point{},
	},
}

func TestProcess(t *testing.T) {
	for _, test := range processTestSets {
		t.Run(test.name, func(t *testing.T) {
			points, _, _, err := Process(jsn, test.i, sink.Point{}, false)
			if err == nil && test.errExpected {
				t.Errorf("error was expected, error was <nil>")
			} else if err != nil && !test.errExpected {
				t.Errorf("no error was expected, error was '%s'", err)
			}
			if !reflect.DeepEqual(points, test.points) {
				t.Log(points)
				t.Log(test.points)
				t.Errorf("points are not as expected")
			}
		})
	}
}
