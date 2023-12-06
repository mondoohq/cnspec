// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"context"
	"errors"
	"os"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v9/shared"
	"go.mondoo.com/cnspec/v9/policy"
	"gocloud.dev/pubsub"
	_ "gocloud.dev/pubsub/awssnssqs"
	"sigs.k8s.io/yaml"
)

var sqsRegex = regexp.MustCompile(`(https:\/\/|http:\/\/)?(sqs)[.][a-z]{2}[-][a-z]{3,}[-][0-9]{1}[.](amazonaws.com)[\/][0-9]{12}[\/]{1}[a-zA-Z0-9-_]*`)

type OutputTarget byte

const (
	NOOP OutputTarget = iota + 1
	LOCAL_FILE
	AWS_SQS
)

type OutputHandler interface {
	WriteReport(ctx context.Context, report *policy.ReportCollection) error
}

type awsSqsHandler struct {
	sqsQueueUrl string
	format      Format
}

type localFileHandler struct {
	file   string
	format Format
}

// default handler, does nothing
type noopHandler struct{}

func (r *noopHandler) WriteReport(ctx context.Context, report *policy.ReportCollection) error {
	return nil
}

func (r *localFileHandler) WriteReport(ctx context.Context, report *policy.ReportCollection) error {
	json, err := convertReport(report, r.format)
	if err != nil {
		return err
	}
	// strip off the file:// prefix from the target
	fileLoc := strings.TrimPrefix(r.file, "file://")

	err = os.WriteFile(fileLoc, json, 0o644)
	if err != nil {
		return err
	}
	log.Info().Str("file", fileLoc).Msg("wrote report to file")
	return nil
}

func (r *awsSqsHandler) WriteReport(ctx context.Context, report *policy.ReportCollection) error {
	// the url may be passed in with a https:// or an http:// prefix, we can trim those
	trimmedUrl := strings.TrimPrefix(r.sqsQueueUrl, "https://")
	trimmedUrl = strings.TrimPrefix(trimmedUrl, "http://")
	topic, err := pubsub.OpenTopic(ctx, "awssqs://"+trimmedUrl)
	if err != nil {
		return err
	}
	defer topic.Shutdown(ctx) //nolint: errcheck
	json, err := convertReport(report, r.format)
	if err != nil {
		return err
	}
	err = topic.Send(ctx, &pubsub.Message{
		Body: json,
	})
	if err != nil {
		return err
	}
	log.Info().Str("url", r.sqsQueueUrl).Msg("sent report to SQS queue")
	return nil
}

func NewOutputHandler(target string, fmt string) (OutputHandler, error) {
	format, ok := Formats[strings.ToLower(fmt)]
	if !ok {
		return nil, errors.New("unknown output format '" + fmt + "'. Available: " + AllFormats())
	}
	typ, err := determineOutputType(target)
	if err != nil {
		return nil, err
	}
	switch typ {
	case LOCAL_FILE:
		return &localFileHandler{file: target, format: format}, nil
	case AWS_SQS:
		return &awsSqsHandler{sqsQueueUrl: target, format: format}, nil
	default:
		return &noopHandler{}, nil
	}
}

// determines the output type based on the provided string. we assume type can be inferred without needing
// extra param to specify the type explicitly
func determineOutputType(target string) (OutputTarget, error) {
	if target == "" {
		return NOOP, nil
	}
	if strings.HasPrefix(target, "file://") {
		return LOCAL_FILE, nil
	}
	if sqsRegex.MatchString(target) {
		return AWS_SQS, nil
	}
	return NOOP, errors.New("could not determine output target type")
}

func convertReport(report *policy.ReportCollection, format Format) ([]byte, error) {
	switch format {
	case YAML:
		return reportToYaml(report)
	// we assume JSON by default if its anything other than explicit YAML
	default:
		return reportToJson(report)
	}
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
