// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"context"
	"os"
	"strings"

	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/utils/iox"
	_ "gocloud.dev/pubsub/awssnssqs"
	_ "gocloud.dev/pubsub/azuresb"
	"sigs.k8s.io/yaml"
)

type HandlerConfig struct {
	Format        string
	OutputTarget  string
	Incognito     bool
	RiskThreshold int
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
		// An OHDF document describes a single asset, so a multi-asset scan needs a
		// file each. Pointing --output-target at a directory asks for exactly that.
		if conf.format == FormatHDF && isDirTarget(config.OutputTarget) {
			return &hdfDirHandler{dir: strings.TrimPrefix(config.OutputTarget, "file://")}, nil
		}
		// OCSF splits its events across one file per event class, so it brings
		// its own handler instead of writing a single stream to one file.
		if conf.format == FormatOcsfJson || conf.format == FormatOcsfParquet {
			return &ocsfFileHandler{target: config.OutputTarget, conf: conf}, nil
		}
		return &localFileHandler{file: config.OutputTarget, conf: conf}, nil
	case AWS_SQS:
		return &awsSqsHandler{sqsQueueUrl: config.OutputTarget, format: conf.format}, nil
	case AZURE_SBUS:
		return &azureSbusHandler{url: config.OutputTarget, format: conf.format}, nil
	case CLI:
		fallthrough
	default:
		res := NewReporter(conf, config.Incognito)
		res.RiskThreshold = config.RiskThreshold
		return res, nil
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

// isDirTarget reports whether an output target names a directory: one that already
// exists, or a path written with a trailing separator to ask for one.
//
// Every format that can write per-asset or per-class files shares this, so
// --output-target means the same thing whichever one is selected. It decides on
// facts (the path is a directory) or explicit intent (a trailing separator) and
// never on a guess: an earlier copy in the OCSF handler also read a
// non-existent extensionless path as a directory, which silently turns an
// ordinary Unix file target like "results" into a directory full of files.
// Anything that does not exist yet and does not end in a separator is a file.
func isDirTarget(target string) bool {
	target = strings.TrimPrefix(target, "file://")
	if target == "" {
		return false
	}
	if strings.HasSuffix(target, "/") || strings.HasSuffix(target, string(os.PathSeparator)) {
		return true
	}
	info, err := os.Stat(target)
	return err == nil && info.IsDir()
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
