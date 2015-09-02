Yarder
================

It cuts logs and then moves those pieces somewhere else

https://en.wikipedia.org/wiki/Yarder

You use it by invoking it with environment variables that tell it what to do. It will
tail a log, output that to another file, gzip it, and upload it to s3

Here is a sample

```
YARDER_LOG_FILE=/Users/stabby/railsboot.txt YARDER_S3_BUCKET=log_bucket YARDER_DURATION=5s YARDER_OUTPUT_FILE=/Users/stabby/blah/test.log YARDER_S3_PATH=logs/stabby go run yarder.go
```

YARDER_LOG_FILE : The log to tail
YARDER_DURATION : The time to tail (format is like "1s" , "5m", or "2h" for 1 second, 5 minutes, or 2 hours respectively)
YARDER_OUTPUT_FILE : The file that will hold the tailed data. Include extensions, but not .tar.gz - this will be added for you
YARDER_S3_BUCKET : The S3 bucket to store things in
YARDER_S3_PATH : Inside of the bucket, the path to keep the cut log file

Roadmap
=========
Improve documentation
Fix whatever bugs probably exist

LICENSE
=========
Apache v2 - See LICENSE
