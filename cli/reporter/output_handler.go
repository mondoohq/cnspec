// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.mondoo.com/cnspec/policy"
	ocsfconvert "go.mondoo.com/cnspec/reports/ocsf/convert"
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

// NewOutputHandlers builds one handler per --output value and returns them as a
// single handler that writes them in the order given. Every format renders the
// same in-memory report, so each extra output costs rendering time only.
//
// config.OutputTarget is the default destination for an output that does not
// name its own with file= or target=. At most one output may use any one destination,
// stdout excepted: several reports printed to stdout appear one after another.
func NewOutputHandlers(formats []string, config HandlerConfig) (OutputHandler, error) {
	if len(formats) == 0 {
		formats = []string{""}
	}

	handlers := make(multiHandler, 0, len(formats))
	used := map[string]string{}
	for _, format := range formats {
		conf, err := ParseConfig(format)
		if err != nil {
			return nil, err
		}
		target := config.OutputTarget
		if conf.target != "" {
			target = conf.target
		}
		if target != "" {
			key := destinationKey(target, conf.targetIsFile)
			if prev, ok := used[key]; ok {
				return nil, fmt.Errorf("outputs %q and %q both write to %s; give each its own %s= or %s=", prev, format, target, OptionFile, OptionTarget)
			}
			used[key] = format
		}

		cur := config
		cur.Format = format
		h, err := newOutputHandler(conf, cur)
		if err != nil {
			return nil, err
		}
		handlers = append(handlers, h)
	}

	if len(handlers) == 1 {
		return handlers[0], nil
	}
	return handlers, nil
}

// destinationKey names the place an output is written, so that two spellings of
// one local path ("out", "./out", "/abs/out", "file://out") compare equal. Queue
// URLs are compared as written.
func destinationKey(target string, isFile bool) string {
	if !isFile && determineOutputType(target) != LOCAL_FILE {
		return target
	}
	path := strings.TrimPrefix(target, "file://")
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return filepath.Clean(path)
}

// multiHandler writes the report to every handler in order. A failing output
// does not stop the ones after it: a scan that cannot reach a queue should still
// leave its SARIF file behind.
type multiHandler []OutputHandler

func (m multiHandler) WriteReport(ctx context.Context, report *policy.ReportCollection) error {
	var errs []error
	for _, h := range m {
		if err := h.WriteReport(ctx, report); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func NewOutputHandler(config HandlerConfig) (OutputHandler, error) {
	conf, err := ParseConfig(config.Format)
	if err != nil {
		return nil, err
	}
	return newOutputHandler(conf, config)
}

func newOutputHandler(conf *PrintConfig, config HandlerConfig) (OutputHandler, error) {
	// file= and target= on the output itself win over --output-target.
	if conf.target != "" {
		config.OutputTarget = conf.target
	}

	typ := determineOutputType(config.OutputTarget)
	if conf.targetIsFile {
		typ = LOCAL_FILE
	}
	switch typ {
	case LOCAL_FILE:
		// An OHDF document describes a single asset, so a multi-asset scan needs a
		// file each. Pointing --output-target at a directory asks for exactly that.
		if conf.format == FormatHDF && isDirTarget(config.OutputTarget) {
			return &hdfDirHandler{dir: strings.TrimPrefix(config.OutputTarget, "file://")}, nil
		}
		// OCSF splits its events across one file per event class whenever it is
		// given a directory, and Parquet has no other form at all.
		if conf.format == FormatOcsfParquet ||
			(conf.format == FormatOcsfJson && isDirTarget(config.OutputTarget)) {
			encoding := ocsfconvert.EncodingJSON
			if conf.format == FormatOcsfParquet {
				encoding = ocsfconvert.EncodingParquet
			}
			return &ocsfDirHandler{
				dir:      strings.TrimPrefix(config.OutputTarget, "file://"),
				encoding: encoding,
				opts:     conf.ocsfOptions(),
			}, nil
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
