# traductio

![traductio](./wordart.png "traductio")

`traductio` reads a JSON data from configurable sources, processes the data using `jq` like syntax and stores
the results in an time series database such as [InfluxDB](https://www.influxdata.com/products/influxdb/) or
[Amazon Timestream](https://aws.amazon.com/timestream/). This can be useful in various contexts, however fetching
data from an [ElasticSearch](https://www.elastic.co/elasticsearch/) and storing aggregates in a performant and
space saving time series is the most obvious use case.

## Installation

To install `traducito` from source you need to have (Go installed)[https://go.dev/doc/install]. With this is place
you can simply run:

```
# go get -u github.com/unprofession-al/traductio
```

To fetch `traductio` as binary navigate to the [releases on GitHub](https://github.com/unprofession-al/traductio/releases)
and download the latest version.

## General Usage

`traductio` can be operated as a command line tool or as an _AWS Lambda_.

When executed __as a simple command line tool__ use the `--help` flag to learn about its usage:

```
# traductio --help
Read data from JSON, process it and store the results in a time series database

Usage:
  traductio [command]

Available Commands:
  help        Help about any command
  run         Performs all steps
  version     Print version info

Flags:
  -c, --cfg string     configuration file path (default "$HOME/traductio.yaml")
  -h, --help           help for traductio
  -v, --vars strings   key:value pairs of variables to be used in the input templates

Use "traductio [command] --help" for more information about a command.
```

You can also run `traductio` as a ___AWS Lambda___. For this you need to create a function (choose the `go1.x` runtime)
and upload the binary to the function. You can use the prepared zip archives of `traducito` which can be found
on GitHub to do so, for example:

```bash
#!/bin/bash
tmp_dir=$(mktemp -d -t traducito-$(date +%Y-%m-%d-%H-%M-%S)-XXXXXXXXXX)
(cd $tmp_dir && curl -L https://github.com/unprofession-al/traductio/releases/download/v0.0.1/traductio_0.0.1_Linux_x86_64.zip -o lambda.zip)
aws --profile [AWS_PROFILE] --region [AWS_REGION] lambda update-function-code --function-name [FUNC_NAME] --zip-file fileb://$tmp_dir/lambda.zip
rm -rf $tmp_dir
```

Using _Amazon EventBridge_ you can then create Rules which executed the Lambda: You need to pass the subcommand and arguments via _Constant
(JSON text)_ in the following form:

```JSON
{
  "command": "run",
  "args": {
    "-c": "s3://bucket/object.yaml",
    "-v": "from:10 days ago,to:1 day ago"  
  }
}
```

## Step by Step

To fetch, process and store the data required `traductio` executes a few steps. Let's have a look into each of
these steps in detail.

> Note that the `--stop-after` flag can be passed to `traductio run` to force the process to stop after a certain
> step. With this you also force `traductio` to print information about the last step executed. This can be useful
> while debugging.

### ReadConfig

Where executing `traductio run` you need to provide the path to a configuration file which (using the `-c` flag)
will be then read in this stop. You can use the following sources: 

When running `traductio run -c ./[local_file_name]` __the local file__ specified is read. Quite unspectacular.

To read a configuration file from __AWS S3__ your can express the value of the `-c` argument using the pattern 
`s3://[bucket]/[path]/[to]/[object]`. The AWS credentials required to read the object stored on S3 can be
provided [all well known ways](https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/).

Where specifying the `-c` argument as a __HTTP/HTTPS__ URL a get request is performed to read the file.

If you are unsure if the file is read properly use `--stop-after ReadConfig` to print the content of the file
read.

### PreFetch

The configuration specifies how the next steps will be performed. The `input` section of the configuration specifies
with which method the actual data to be processed will be read. The section is processed as a
[Go template](https://pkg.go.dev/text/template). In the PreFetch step these templated portions of the configuration
are rendered using the values specified via the `-v key1:value1,key2:value2,...` argument.

For example lets assume the input section of the configuration files looks like this:

```yaml
--- traductio.yaml
input:
  url: s3://{{.bucket}}/{{.file}}
...
```

By executing the `PreFetch` step the text is properly substituted. Note that by specifying `--stop-after PreFetch`
you can inspect the rendered (prefetched) version of the file:

```
# traductio run -c traductio.yaml -v bucket:example,file:data.json --stop-after PreFetch
--- traductio.yaml
input:
  url: s3://bucket/data.json
```

Apparently the data to be processed is read from an S3 bucket. Simple... 

However in this particular example the benefit of the _templated input_ is not exactly clear. Let's look into a
second example. Here we do not just fetch object from S3 but rather query an ElasticSearch server:

```yaml
--- 
input:
  url: https://elasticsearch.example.com/access-logs-*/_doc/_search
  headers:
    Authorization: ApiKey {{.apikey}}
    Content-Type: application/json
  method: GET
  body: |
    {
        "size": 0,
        "query": {
            "bool": {
                "filter": {
                    "range": {
                        "@timestamp": {
                            "lt": "{{unixMilliTimestamp .to}}",
                            "gt": "{{unixMilliTimestamp .from}}",
                            "format": "epoch_millis"
    ...
```

Again, running with the help of `--stop-after PreFetch` we can inspect the rendered results:

```
# traductio run -c traductio.yaml -v apikey:S3cr3T,from:10 days ago,to:1 day ago --stop-after PreFetch 
--- 
input:
  url: https://elasticsearch.example.com/access-logs-*/_doc/_search
  headers:
    Authorization: ApiKey S3cr3T
    Content-Type: application/json
  method: GET
  body: |
    {
        "size": 0,
        "query": {
            "bool": {
                "filter": {
                    "range": {
                        "@timestamp": {
                            "lt": "1645225200000",
                            "gt": "1644447600000",
                            "format": "epoch_millis"
    ...

```

In this example we have avoided that the `ApiKey` is stored in the configuration file. Furthermore we allow
`traductio` to continuously process data (for example in a cron job) by specifying relative timestamps which are
then rendered into the request body as proper timestamps.

Note that for the timestamps the values on the command line have been passed in a human processable manner and the
expanded using the `unixMilliTimestamp` function. To pass timestamps your can either use the
[RFC3330](https://datatracker.ietf.org/doc/html/rfc3339) format or a [human readable expression](https://github.com/tj/go-naturaldate).
To further process these values the functions `unixTimestamp` and `unixMilliTimestamp` can be used in the template.

### Fetch

Fetch finally reads the data specified in the input section. As with the option `-c` data can be read
via HTTP/S, from S3 or from a local file by specifying the `input.url` portion accordingly. For HTTP/S
requests the exact request can be described in the configuration:

```yaml
---
input:
  url: ...
  headers: ...
    Content-Type: application/json
    Key: Value
  method: GET
  body: { "some": "json" }
  http_expect_status: 200
...
```

With `--stop-after Fetch` you can force `traductio` to exit and print the results of the request, e.g. the data
to be processed.

### Validate

In some cases the data fetched holds some information whether the request should be processed further. For example
`ElasticSearch` provides information on how many shards where successfully queried and how many have failed. I case
of failed shards the data presented might me incomplete, further processing would lead to wrong results.

in the validate section these information can be analyzed. For the case described above the following validator does
the trick:

```yaml
validators:
  - selector: "._shards.failed"
    expect: "0"
```

Note that the `selector` is a [`jq` like](https://github.com/itchyny/gojq) expression. To test these expressions you
can use [`gojq`](https://github.com/itchyny/gojq) or [`jq`](https://stedolan.github.io/jq/) that matter:

```
# traductio run -c ... --stop-after Fetch | jq '._shards.failed'
0
```

### Process

In this step the points to be fed to the time series database will be constructed. `traductio` expects the data returned
in the `Fetch` step to have a tree structure (eg. the sort of data structure you would expect from querying `ElasticSearch`).
Here's a simple example; lets assume we have received the following data in the `Fetch` step:

```json
{
  ...,
  "aggregations": {
    "over_time": {
      "buckets": [
        {
          "key_as_string": "2022-02-17T00:00:00.000+01:00",
          "key": 1645052400000,
          "doc_count": 26228,
          "by_domain": {
            "doc_count_error_upper_bound": 0,
            "sum_other_doc_count": 0,
            "buckets": [
              {
                "key": "print.geo.admin.ch",
                "doc_count": 26228,
                "uniq_users": {
                  "value": 1727
                },
                "bytes_sent": {
                  "value": 14225629166
                }
              }
            ]
          }
        },
        {
          "key_as_string": "2022-02-18T00:00:00.000+01:00",
          "key": 1645138800000,
          "doc_count": 24489,
          "by_domain": {
            "doc_count_error_upper_bound": 0,
            "sum_other_doc_count": 0,
            "buckets": [
              {
                "key": "print.geo.admin.ch",
                "doc_count": 24489,
                "uniq_users": {
                  "value": 1663
                },
                "bytes_sent": {
                  "value": 11819885976
                }
              }
            ]
          }
        }
      ]
    }
  }
}

```

To construct the points to be stored we need to navigate thru that tree structure in an iterative manner and
gather the data required along the way. Again this is done using [`jq` like](https://github.com/itchyny/gojq)
expression. In that particular case a proper configuration for the processor would look like this:

```yaml
---
...
process:
  no_trim: false
  iterator:
    selector: .aggregations.over_time.buckets[]
    time:
      selector: .key
      format: unixMilliTimestamp
    iterator:
      selector: .by_domain.buckets[]
      tags:
        domain: .key
      values:
        request_count: .doc_count
        bytes_sent: .bytes_sent.value
        uniq_users: .uniq_users.value
...
```

> Note that `traductio` will trim all points with the oldest and the newest timestamp available in the
> data set available. This is because these border area data are often not complete when (for example if
> data is gathered on a hourly base and the time frame considered in the ElasticSearch query does not
> exactly start at minute :00 the data will not be complete for this first hour). To disable this behaviour
> the `no_trim` option can be set to `true`.

To build a complete point in a time series three types of values are required: A _time stamp_ which is extracted using
the `time` portion, _tags_ (also known as _dimensions_) which are usually string values, and the values at that point
in time reflected by a number.

Executing `traductio` with the  `--stop-after Process` flag will print the points extracted from the
raw data as CSV:

```csv
time,domain,request_count,bytes_sent,uniq_users
2022-02-16 00:00:00 +0100 CET,www.example.com,25170.000000,13330701559.000000,1754.000000
2022-02-17 00:00:00 +0100 CET,www.example.com,26228.000000,14225629166.000000,1727.000000
```

### Store

As a last step the data must be persisted into a time series database. `traductio` supports two types of databases:

_AWS Timestream_ requires the following information in the `output section`:

```yaml
---
...
output:
  kind: timestream
  connection:
    region: [for example eu-west-1]
    db: [name of the database]
    series: [name of the series]
```

_InfluxDB v2_ needs the following fields to be provided:

```yaml
---
...
output:
  kind: timestream
  connection:
    addr: [for example http://localhost:8086]
    token: [access toker]
    org: [name of the org]
    bucket: [name of bucket]
    series: [name of the series]
```

> Note that when storing data to the database an _upsert_ will be performed. This means that old data
> with the same timestamps and tags/dimensions will be replaced.

