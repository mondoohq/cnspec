// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"context"
	"errors"
	"strings"

	"go.mondoo.com/cnquery/v9/shared"
	"go.mondoo.com/cnspec/v9/policy"
	_ "gocloud.dev/pubsub/awssnssqs"
	"sigs.k8s.io/yaml"
)

type ReportConfig struct {
	Format       string
	OutputTarget string
	Incognito    bool
}

type OutputTarget byte

const (
	CLI OutputTarget = iota + 1
	LOCAL_FILE
	AWS_SQS
)

type OutputHandler interface {
	WriteReport(ctx context.Context, report *policy.ReportCollection) error
}

func NewOutputHandler(config ReportConfig) (OutputHandler, error) {
	format, ok := Formats[strings.ToLower(config.Format)]
	if !ok {
		return nil, errors.New("unknown output format '" + config.Format + "'. Available: " + AllFormats())
	}
	typ := determineOutputType(config.OutputTarget)
	switch typ {
	case LOCAL_FILE:
		return &localFileHandler{file: config.OutputTarget, format: format}, nil
	case AWS_SQS:
		return &awsSqsHandler{sqsQueueUrl: config.OutputTarget, format: format}, nil
	case CLI:
		fallthrough
	default:
		return NewReporter(format, config.Incognito), nil
	}
}

// determines the output type based on the provided string. we assume type can be inferred without needing
// extra param to specify the type explicitly
func determineOutputType(target string) OutputTarget {
	// we fall back to CLI reporting, default behavior
	if target == "" {
		return CLI
	}
	if sqsRegex.MatchString(target) {
		return AWS_SQS
	}

	return LOCAL_FILE
}

func reportToYaml(report *policy.ReportCollection) ([]byte, error) {
	json, err := reportToJson(report)
	if err != nil {
		return nil, err
	}
	yaml, err := yaml.JSONToYAML(json)
	if err != nil {
		return nil, err
	}
	return yaml, nil
}

func reportToJson(report *policy.ReportCollection) ([]byte, error) {
	raw := bytes.Buffer{}
	writer := shared.IOWriter{Writer: &raw}
	err := ReportCollectionToJSON(report, &writer)
	if err != nil {
		return nil, err
	}
	return raw.Bytes(), nil
}
