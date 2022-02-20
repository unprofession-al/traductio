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

### Validate

### Process

### Store

#
