package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/minio/minio-go"
)

var errChecksumMismatch = errors.New("checksum mismatch")

const usageTemplate = `NAME:
  service-loader

  Loads binaries from a amazon s3 bucket or minio and validates it against
  the sha256 checksum included along side it.

USAGE:
  service-loader [options] <APP> <TAG>

OPTIONS:
  -help
    	show help`

func usage() {
	fmt.Fprintln(os.Stderr, usageTemplate)
	flag.PrintDefaults()
}

func init() {
	flag.Usage = usage
}

func defaultOS(key, fallback string) string {
	if env := os.Getenv(key); env != "" {
		return env
	}

	return fallback
}

func exit(err error, printUsage bool) {
	fmt.Println(err.Error())

	if printUsage {
		usage()
	}

	os.Exit(1)
}

func main() {
	var bucket = flag.String("bucket", "", "S3/minio bucket where the releases exist")
	var id = flag.String("id", defaultOS("AWS_ACCESS_KEY_ID", ""), "AWS_ACCESS_KEY_ID to authenticate to the aws/minio service, defaults to env variable")
	var key = flag.String("key", defaultOS("AWS_SECRET_ACCESS_KEY", ""), "AWS_SECRET_ACCESS_KEY to authenticate to the aws/minio service, defaults to env variable")
	var endpoint = flag.String("endpoint", "s3.amazonaws.com", "endpoint for the aws/minio service")
	var sha = flag.String("sha", "", "the sha256 to validate against, defaults to loading from aws/minio")

	flag.Parse()

	if *bucket == "" {
		exit(errors.New("-bucket is required"), true)
	}

	if *id == "" {
		exit(errors.New("-id is required"), true)
	}

	if *key == "" {
		exit(errors.New("-key is required"), true)
	}

	args := flag.Args()

	if len(args) < 1 {
		exit(errors.New("APP and TAG are required"), true)
	} else if len(args) < 2 {
		exit(errors.New("TAG is required"), true)
	}

	var app = args[0]
	var tag = args[1]

	err := run(runOps{
		app:      app,
		tag:      tag,
		bucket:   *bucket,
		id:       *id,
		key:      *key,
		os:       defaultOS("GOOS", runtime.GOOS),
		arch:     defaultOS("GOARCH", runtime.GOARCH),
		endpoint: *endpoint,
		sha:      *sha,
	})
	if err != nil {
		exit(err, false)
	}
}

func generateSha256(buf io.Reader) (string, error) {
	h := sha256.New()

	if _, err := io.Copy(h, buf); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

type runOps struct {
	app      string // application to load
	tag      string // tag to reference
	endpoint string // aws/minio endpoint
	bucket   string // aws/minio bucket where archives exist
	id       string // AWS_ACCESS_KEY_ID to access aws/minio
	key      string // AWS_SECRET_ACCESS_KEY to access aws/minio
	os       string // os to use to load the archive with
	arch     string // arch to use to load the archive with
	sha      string // optional sha to compare with instead of loading the file
}

func download(client *minio.Client, bucket, fileName string) (*bytes.Buffer, error) {
	file, err := client.GetObject(bucket, fileName)
	if err != nil {
		return nil, err
	}

	var buf = bytes.NewBuffer(nil)

	if _, err := io.Copy(buf, file); err != nil {
		return nil, err
	}

	return buf, nil
}

func run(opts runOps) error {
	minioClient, err := minio.New(opts.endpoint, opts.id, opts.key, true)
	if err != nil {
		return err
	}

	fileName := fmt.Sprintf("%s_%s_%s_%s", opts.app, opts.tag, opts.os, opts.arch)

	tarKey := fileName + ".tar.gz"

	fmt.Printf("## Downloading Archive: %s/%s\n", opts.bucket, tarKey)

	archive, err := download(minioClient, opts.bucket, tarKey)
	if err != nil {
		return fmt.Errorf("Can't download archive from %s/%s", opts.bucket, tarKey)
	}

	fmt.Printf("## Done\n\n")

	fmt.Printf("## Generating Checksum\n")

	localChecksum, err := generateSha256(archive)
	if err != nil {
		return err
	}

	fmt.Printf("## Done\n\n")

	if opts.sha == "" {

		checksumKey := fileName + ".sha256"

		fmt.Printf("## Downloading Checksum: %s/%s\n", opts.bucket, checksumKey)

		remoteSha256, err := download(minioClient, opts.bucket, checksumKey)
		if err != nil {
			return fmt.Errorf("Can't download checksum from %s/%s", opts.bucket, checksumKey)
		}

		opts.sha = strings.Fields(remoteSha256.String())[0]

		fmt.Printf("## Done\n\n")
	}

	fmt.Printf("## Comparing checksums\n")

	if opts.sha != localChecksum {
		fmt.Printf("## FAIL\n\n")
		return errChecksumMismatch
	}

	fmt.Printf("## PASS\n\n")

	fmt.Printf("## Saving Archive to %s\n", tarKey)

	archiveFile, err := os.Create(tarKey)
	if err != nil {
		return err
	}

	if _, err = io.Copy(archiveFile, archive); err != nil {
		return err
	}

	fmt.Printf("## Done\n")

	return nil
}
