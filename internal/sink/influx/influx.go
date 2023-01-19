package influx

import (
	"context"
	"fmt"
	"traductio/internal/sink"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

func init() {
	sink.Register("influx", setup)
}

type Influx struct {
	Name   string
	Client influxdb2.Client
	Bucket string
	Org    string
	Series string
}

func setup(config map[string]string) (sink.Sink, error) {
	i := Influx{}

	var found bool
	var addr, token, org, bucket, series string

	if addr, found = config["addr"]; !found || addr == "" {
		return i, fmt.Errorf("sink requires field 'addr' to be set")
	}
	if token, found = config["token"]; !found || token == "" {
		return i, fmt.Errorf("sink requires field 'token' to be set")
	}
	if org, found = config["org"]; !found || org == "" {
		return i, fmt.Errorf("sink requires field 'bucket' to be set")
	}
	if bucket, found = config["bucket"]; !found || bucket == "" {
		return i, fmt.Errorf("sink requires field 'bucket' to be set")
	}
	if series, found = config["series"]; !found || series == "" {
		return i, fmt.Errorf("sink requires field 'series' to be set")
	}

	c := influxdb2.NewClientWithOptions(addr, token, influxdb2.DefaultOptions().SetBatchSize(20))

	i = Influx{
		Name: fmt.Sprintf("influx:%s:%s", config["bucket"], config["series"]),
		Client: c,
		Org:    org,
		Bucket: bucket,
		Series: series,
	}
	return i, nil
}

func (i Influx) Write(points []sink.Point) error {
	writeAPI := i.Client.WriteAPIBlocking(i.Org, i.Bucket)

	if len(points) < 1 {
		return fmt.Errorf("no points to be written")
	}

	for _, p := range points {
		values := map[string]interface{}{}
		for k, v := range p.Values {
			values[k] = v
		}
		pt := influxdb2.NewPoint(i.Series, p.Tags, values, p.Timestamp)
		err := writeAPI.WritePoint(context.Background(), pt)
		if err != nil {
			return err
		}
	}

	return nil
}

func (i Influx) Close() {
	i.Client.Close()
}

func (i Influx) GetName() string {
	return i.Name
}