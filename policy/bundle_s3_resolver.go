// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"io"
	"strings"

	awsconfigv2 "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog/log"
)

type S3GetObjectAPI interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	ListObjects(ctx context.Context, params *s3.ListObjectsInput, optFns ...func(*s3.Options)) (*s3.ListObjectsOutput, error)
}

type s3BundleResolver struct {
	s3Client S3GetObjectAPI
}

func defaultS3BundleResolver() *s3BundleResolver {
	return &s3BundleResolver{}
}

func NewS3BundleResolver(api S3GetObjectAPI) *s3BundleResolver {
	return &s3BundleResolver{s3Client: api}
}

func (r *s3BundleResolver) Load(ctx context.Context, path string) (*Bundle, error) {
	// if we have no explicitly provided s3 client, we need to create one. we can optimize this later on by ensuring we require a client
	// when creating the s3 bundle resolver
	if r.s3Client == nil {
		cfg, err := awsconfigv2.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, err
		}
		r.s3Client = s3.NewFromConfig(cfg)
	}
	trimmed := strings.TrimPrefix(path, "s3://")
	if ok, bucket, key := isS3Key(trimmed); ok {
		return r.fetchBundleFromS3(ctx, bucket, key)
	}

	// else we're looking at a bucket so we need to iterate over all the files inside
	files, err := r.s3Client.ListObjects(ctx, &s3.ListObjectsInput{Bucket: &trimmed})
	if err != nil {
		return nil, err
	}
	mergedBundle := &Bundle{}
	resolvedNames := []string{}
	for _, file := range files.Contents {
		key := *file.Key
		if strings.HasSuffix(key, ".mql.yaml") || strings.HasSuffix(key, ".mql.yml") {
			resolvedNames = append(resolvedNames, key)
		}
	}
	for _, file := range resolvedNames {
		bundle, err := r.fetchBundleFromS3(ctx, trimmed, file)
		if err != nil {
			return nil, err
		}
		mergedBundle = Merge(mergedBundle, bundle)
	}
	return mergedBundle, nil
}

func (r *s3BundleResolver) IsApplicable(path string) bool {
	return strings.HasPrefix(path, "s3://")
}

// basic check that validates if the provided path is a s3 key. it returns the bucket and the key that can be used to fetch the object
// an s3 key is defined as a path that contains more than one slash, e.g. my-bucket/my-key
func isS3Key(path string) (bool, string, string) {
	parts := strings.Split(path, "/")
	if len(parts) > 1 {
		return true, parts[0], strings.Join(parts[1:], "/")
	}
	return false, "", ""
}

func (r *s3BundleResolver) fetchBundleFromS3(ctx context.Context, bucket string, key string) (*Bundle, error) {
	resp, err := r.s3Client.GetObject(ctx, &s3.GetObjectInput{Bucket: &bucket, Key: &key})
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	log.Debug().Str("bucket", bucket).Str("key", key).Msg("s3>loaded bundle file from s3")
	return BundleFromYAML(data)
}
