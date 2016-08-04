# service-loader

Downloads an application binary inside an archive matching
the format `{APP}_{TAG}_{GOOS}_{GOARCH}.tag.gz` and validates it against either
the `-sha` flag or the file in the format `{APP}_{TAG}_{GOOS}_{GOARCH}.sha256`
which contains the sha256 hash of the archive as generated by the `sha256sum`
utility.

## Installation

Check out the releases page for a binary version.

Source:

```
go get github.com/wyattjoh/service-loader
```

## Usage

```
NAME:
  service-loader

  Loads binaries from a amazon s3 bucket or minio and validates it against
  the sha256 checksum included along side it.

USAGE:
  service-loader [options] <APP> <TAG>

OPTIONS:
  -help
    	show help
  -bucket string
    	S3/minio bucket where the releases exist
  -endpoint string
    	endpoint for the aws/minio service (default "s3.amazonaws.com")
  -id string
    	AWS_ACCESS_KEY_ID to authenticate to the aws/minio service, defaults to env variable
  -key string
    	AWS_SECRET_ACCESS_KEY to authenticate to the aws/minio service, defaults to env variable
  -sha string
    	the sha256 to validate against, defaults to loading from aws/minio
```

## License

MIT
