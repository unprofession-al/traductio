# traductio

![traductio](./wordart.png "traductio")

`traductio` reads a JSON file from a HTTP endpoint, S3 or just a plain old local file,
processes the data and stores the results in an time series database such as
[InfluxDB](https://www.influxdata.com/products/influxdb/) or
[Amazon Timestream](https://aws.amazon.com/timestream/). This can be useful in various contexts,
however fetching data from an [ElasticSearch](https://www.elastic.co/elasticsearch/)
and storing aggregates in a performant and space saving time series is the
most obvious use case.

