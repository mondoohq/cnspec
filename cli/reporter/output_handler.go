// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"context"

	"go.mondoo.com/cnquery/v11/utils/iox"
	"go.mondoo.com/cnspec/v11/policy"
	_ "gocloud.dev/pubsub/awssnssqs"
	_ "gocloud.dev/pubsub/azuresb"
	"sigs.k8s.io/yaml"
)

type HandlerConfig struct {
	Format         string
	OutputTarget   string
	Incognito      bool
	ScoreThreshold int
}

type OutputTarget byte

const (
	CLI OutputTarget = iota + 1
	LOCAL_FILE
	AWS_SQS
	AZURE_SBUS
)

type OutputHandler interface {
	WriteReport(ctx context.Context, report *policy.ReportCollection) error
}

func NewOutputHandler(config HandlerConfig) (OutputHandler, error) {
	conf, err := ParseConfig(config.Format)
	if err != nil {
		return nil, err
	}

	typ := determineOutputType(config.OutputTarget)
	switch typ {
	case LOCAL_FILE:
		return &localFileHandler{file: config.OutputTarget, conf: conf}, nil
	case AWS_SQS:
		return &awsSqsHandler{sqsQueueUrl: config.OutputTarget, format: conf.format}, nil
	case AZURE_SBUS:
		return &azureSbusHandler{url: config.OutputTarget, format: conf.format}, nil
	case CLI:
		fallthrough
	default:
		return NewReporter(conf, config.Incognito, config.ScoreThreshold), nil
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
	if sbusRegex.MatchString(target) {
		return AZURE_SBUS
	}

	return LOCAL_FILE
}

func reportToYamlV1(report *policy.ReportCollection) ([]byte, error) {
	json, err := reportToJsonV1(report)
	if err != nil {
		return nil, err
	}
	yaml, err := yaml.JSONToYAML(json)
	if err != nil {
		return nil, err
	}
	return yaml, nil
}

func reportToJsonV1(report *policy.ReportCollection) ([]byte, error) {
	raw := bytes.Buffer{}
	writer := iox.IOWriter{Writer: &raw}
	err := ConvertToJSON(report, &writer)
	if err != nil {
		return nil, err
	}
	return raw.Bytes(), nil
}

func reportToYamlV2(report *policy.ReportCollection) ([]byte, error) {
	json, err := reportToJsonV2(report)
	if err != nil {
		return nil, err
	}
	yaml, err := yaml.JSONToYAML(json)
	if err != nil {
		return nil, err
	}
	return yaml, nil
}

func reportToJsonV2(report *policy.ReportCollection) ([]byte, error) {
	r, err := ConvertToProto(report)
	if err != nil {
		return nil, err
	}

	return r.ToJSON()
}
