package sink

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"reflect"
	"text/tabwriter"
	"time"
)

type Point struct {
	Timestamp time.Time
	Tags      map[string]string
	Values    map[string]float64
}

func PointAvg(points []Point, samples int) []Point {
	var measurements []Point
	for _, point := range points {
		found := false
		for _, measurement := range measurements {
			if measurement.Timestamp == point.Timestamp && reflect.DeepEqual(measurement.Tags, point.Tags) {
				for key, value := range point.Values {
					add := value
					pre := measurement.Values[key]
					measurement.Values[key] = pre + add
				}
				found = true
				break
			}
		}
		if !found {
			measurements = append(measurements, point.Copy())
		}
	}

	for index, measurement := range measurements {
		for key, value := range measurement.Values {
			pre := value
			measurements[index].Values[key] = pre / float64(samples)
		}
	}

	return measurements
}

func TrimPoints(points []Point) ([]Point, time.Time, time.Time) {
	earliest := time.Now()
	var latest time.Time
	for _, p := range points {
		if p.Timestamp.Before(earliest) {
			earliest = p.Timestamp
		}
		if p.Timestamp.After(latest) {
			latest = p.Timestamp
		}
	}

	var out []Point
	for _, p := range points {
		if p.Timestamp != earliest && p.Timestamp != latest {
			out = append(out, p)
		}
	}

	return out, earliest, latest
}

func (p Point) String() string {
	out := new(bytes.Buffer)
	timestamp := p.Timestamp.Format("2006-01-02 15:04:05")

	const padding = 1

	var valuesStr []string
	for key, val := range p.Values {
		valuesStr = append(valuesStr, fmt.Sprintf("%s: %v", key, val))
	}

	var tagsStr []string
	for key, val := range p.Tags {
		tagsStr = append(tagsStr, fmt.Sprintf("%s: %s", key, val))
	}

	var iterations int

	if len(tagsStr) > len(valuesStr) {
		iterations = len(tagsStr)
	} else {
		iterations = len(valuesStr)
	}

	w := tabwriter.NewWriter(out, 0, 0, padding, ' ', tabwriter.Debug)
	fmt.Fprintf(w, "@%s\t Tags\t Values\n", timestamp)
	for i := 0; i < iterations; i++ {
		tag := ""
		if len(tagsStr) > i {
			tag = tagsStr[i]
		}
		value := ""
		if len(valuesStr) > i {
			value = valuesStr[i]
		}
		fmt.Fprintf(w, "\t %s\t %s\n", tag, value)
	}
	w.Flush()

	return string(out.String())
}

func (p Point) Copy() Point {
	tags := make(map[string]string)
	for k, v := range p.Tags {
		tags[k] = v
	}

	values := make(map[string]float64)
	for k, v := range p.Values {
		values[k] = v
	}

	c := Point{
		Timestamp: p.Timestamp,
		Tags:      tags,
		Values:    values,
	}
	return c
}

func (p Point) IsEmpty() bool {
	var unsetTime time.Time
	if len(p.Values) > 0 && p.Timestamp != unsetTime {
		return false
	}
	return true
}

func PointsAsCSV(points []Point, delimiter string) ([]byte, error) {
	var err error
	var out bytes.Buffer

	writer := csv.NewWriter(&out)
	if writer.Comma, err = asSingleRune(delimiter); err != nil {
		return []byte{}, err
	}

	cols := map[string]bool{
		"time": true,
	}

	data := []map[string]string{}

	for _, p := range points {
		row := map[string]string{
			"time": p.Timestamp.String(),
		}

		for k, v := range p.Tags {
			cols[k] = true
			row[k] = v

		}

		for k, v := range p.Values {
			cols[k] = true
			row[k] = fmt.Sprintf("%f", v)
		}

		data = append(data, row)
	}

	table := [][]string{}

	// write header to table
	header := []string{}
	for columnName := range cols {
		header = append(header, columnName)
	}
	table = append(table, header)

	// fill data rows
	for _, rowData := range data {
		row := []string{}
		for _, colName := range header {
			if v, ok := rowData[colName]; ok {
				row = append(row, v)
			} else {
				row = append(row, "")
			}
		}
		table = append(table, row)
	}

	err = writer.WriteAll(table)
	return out.Bytes(), err
}

func asSingleRune(in string) (rune, error) {
	var out rune
	if len(in) != 1 {
		return out, fmt.Errorf("separator must be exactly one character")
	}
	out = []rune(in)[0]
	return out, nil
}
