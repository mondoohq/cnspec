// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/internal/reportfixture"
	"go.mondoo.com/cnspec/reports/ocsf"
)

func TestOutputHandlerAwsSqs(t *testing.T) {
	sqsUrls := []string{
		"https://sqs.us-east-1.amazonaws.com/123456789012/MyQueue",
		"http://sqs.us-east-1.amazonaws.com/123456789012/MyQueue",
		"https://sqs.eu-central-1.amazonaws.com/123456789012/MyQueue",
		"http://sqs.eu-central-1.amazonaws.com/123456789012/MyQueue",
		"sqs.us-east-1.amazonaws.com/123456789012/MyQueue",
		"sqs.eu-central-1.amazonaws.com/123456789012/MyQueue",
	}

	for i, sqsUrl := range sqsUrls {
		rep, err := NewOutputHandler(HandlerConfig{Format: "JSON", OutputTarget: sqsUrl})
		require.NoError(t, err, i)
		require.IsType(t, &awsSqsHandler{}, rep, i)
	}
}

func TestOutputHandlerAzureServiceBusSqs(t *testing.T) {
	sbusUrls := []string{
		"https://my-sbus.servicebus.windows.net/my-queue",
		"http://my-sbus.servicebus.windows.net/my-queue",
		"my-sbus.servicebus.windows.net/my-queue",
	}

	for i, sqsUrl := range sbusUrls {
		rep, err := NewOutputHandler(HandlerConfig{Format: "JSON", OutputTarget: sqsUrl})
		require.NoError(t, err, i)
		require.IsType(t, &azureSbusHandler{}, rep, i)
	}
}

func TestOutputHandlerFileLocal(t *testing.T) {
	fileTargets := []string{
		"file:///root/test",
		"file:///root/test.json",
		"file://root/test.json",
		"/root/test.json",
		"test.json",
	}

	for i, f := range fileTargets {
		rep, err := NewOutputHandler(HandlerConfig{Format: "JSON", OutputTarget: f})
		require.NoError(t, err, i)
		require.IsType(t, &localFileHandler{}, rep, i)
	}
}

func TestCliReporter(t *testing.T) {
	rep, err := NewOutputHandler(HandlerConfig{})
	require.NoError(t, err)
	require.IsType(t, &Reporter{}, rep)
}

// TestIsDirTarget pins the one definition of "--output-target names a directory".
//
// Every format that can write per-asset or per-class files shares this helper, so
// the flag has to mean the same thing for all of them. A second copy in the OCSF
// handler used to disagree on exactly one row -- the marked one below -- by
// reading a non-existent extensionless path as a directory. That guess turns an
// ordinary Unix file target like "results" into a directory full of files, so the
// rule is facts (it exists and is a directory) or explicit intent (a trailing
// separator), never an extension heuristic.
func TestIsDirTarget(t *testing.T) {
	dir := t.TempDir()

	existingFile := filepath.Join(dir, "report.json")
	require.NoError(t, os.WriteFile(existingFile, []byte("{}"), 0o600))

	existingDirNoExt := filepath.Join(dir, "results")
	require.NoError(t, os.Mkdir(existingDirNoExt, 0o755))

	tests := []struct {
		name   string
		target string
		want   bool
	}{
		{"empty target is not a directory", "", false},
		{"trailing slash asks for one", filepath.Join(dir, "out") + "/", true},
		{"trailing slash wins over a file extension", filepath.Join(dir, "out.json") + "/", true},
		{"the file:// prefix is stripped before the stat", "file://" + existingDirNoExt, true},
		{"the file:// prefix on an existing file", "file://" + existingFile, false},
		{"existing directory", existingDirNoExt, true},
		{"existing file", existingFile, false},
		// The row the two copies disagreed on.
		{"non-existent extensionless path is a file", filepath.Join(dir, "does-not-exist"), false},
		{"non-existent path with an extension is a file", filepath.Join(dir, "nope.json"), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, isDirTarget(tc.target))
		})
	}
}

// TestOutputHandlerOcsfFindingsDetection walks the whole detection path: the
// option string a user types, through the handler --output-target picks, to the
// events on disk.
//
// The narrower tests cover the converter and the file handler with a config built
// in Go. Nothing else proves that `-o ocsf-json,ocsf-findings=detection` parses,
// routes to the OCSF file handler, and lands class 2004 rather than 2003. An HEC
// output target used to be a second way to reach class 2004, so with it gone this
// is the only path there and wants a test that spans the whole of it.
func TestOutputHandlerOcsfFindingsDetection(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "out") + string(os.PathSeparator)

	handler, err := NewOutputHandler(HandlerConfig{
		Format:       "ocsf-json,ocsf-findings=detection",
		OutputTarget: dir,
	})
	require.NoError(t, err)
	require.IsType(t, &ocsfDirHandler{}, handler)
	require.NoError(t, handler.WriteReport(t.Context(), reportfixture.Sample()))

	require.NoFileExists(t, filepath.Join(dir, ocsf.ClassComplianceFinding+".jsonl"),
		"a check is reported in one class, and detection was the one asked for")

	raw, err := os.ReadFile(filepath.Join(dir, ocsf.ClassDetectionFinding+".jsonl"))
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	require.Len(t, lines, 3, "one detection finding per reporting check")
	for _, line := range lines {
		var event map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &event))
		assert.EqualValues(t, ocsf.ClassUIDDetectionFinding, event["class_uid"])
		// The attributes class 2004 carries and 2003 has nowhere to put, which is
		// what the option exists for.
		assert.Contains(t, event, "risk_level_id")
		assert.Contains(t, event["finding_info"], "analytic")
		// Class 2004 has no compliance object, so the framework mappings travel in
		// unmapped instead.
		assert.NotContains(t, event, "compliance")
	}
}

// TestOutputHandlerOcsfTargets pins which handler each OCSF target reaches, which
// is the whole of what cli/reporter decides about this format.
//
// ocsf-json writes every class into one newline-delimited stream unless the
// target names a directory, and "names a directory" is isDirTarget for every
// format alike -- an extensionless path that does not exist is a file, not a
// directory to fill with per-class files. ocsf-parquet has no single-file form,
// so it takes the directory branch whatever the target looks like.
func TestOutputHandlerOcsfTargets(t *testing.T) {
	dir := t.TempDir()
	existingDir := filepath.Join(dir, "events")
	require.NoError(t, os.Mkdir(existingDir, 0o755))

	tests := []struct {
		name   string
		format string
		target string
		want   OutputHandler
	}{
		{"ocsf-json to a file", "ocsf-json", filepath.Join(dir, "report.jsonl"), &localFileHandler{}},
		{"ocsf-json to an extensionless path", "ocsf-json", filepath.Join(dir, "results"), &localFileHandler{}},
		{"ocsf-json to an existing directory", "ocsf-json", existingDir, &ocsfDirHandler{}},
		{"ocsf-json to a trailing separator", "ocsf-json", filepath.Join(dir, "out") + "/", &ocsfDirHandler{}},
		{"ocsf-parquet is always a directory", "ocsf-parquet", filepath.Join(dir, "report.parquet"), &ocsfDirHandler{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler, err := NewOutputHandler(HandlerConfig{Format: tc.format, OutputTarget: tc.target})
			require.NoError(t, err)
			require.IsType(t, tc.want, handler)
		})
	}
}

// TestOutputHandlerOcsfJsonSingleFile walks the file branch end to end: one
// stream, every class in it.
func TestOutputHandlerOcsfJsonSingleFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.jsonl")

	handler, err := NewOutputHandler(HandlerConfig{Format: "ocsf-json", OutputTarget: "file://" + path})
	require.NoError(t, err)
	require.NoError(t, handler.WriteReport(t.Context(), reportfixture.Sample()))

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	assert.Len(t, lines, 4, "3 checks + 1 inventory event, all in one stream")
}

// TestOutputHandlersWriteInOrder walks several outputs end to end: each lands at
// its own destination, and file= on an output wins over --output-target.
func TestOutputHandlersWriteInOrder(t *testing.T) {
	dir := t.TempDir()
	sarifPath := filepath.Join(dir, "results.sarif")
	jsonPath := filepath.Join(dir, "report.json")

	handler, err := NewOutputHandlers(
		[]string{"sarif,file=" + sarifPath, "json"},
		HandlerConfig{OutputTarget: jsonPath},
	)
	require.NoError(t, err)
	require.IsType(t, multiHandler{}, handler)
	require.NoError(t, handler.WriteReport(t.Context(), reportfixture.Sample()))

	raw, err := os.ReadFile(sarifPath)
	require.NoError(t, err)
	var sarifDoc map[string]any
	require.NoError(t, json.Unmarshal(raw, &sarifDoc))
	assert.Equal(t, "2.1.0", sarifDoc["version"])

	raw, err = os.ReadFile(jsonPath)
	require.NoError(t, err)
	assert.True(t, json.Valid(raw), "--output-target receives the output without file=")
}

func TestOutputHandlersSingleOutputUnchanged(t *testing.T) {
	handler, err := NewOutputHandlers([]string{"json"}, HandlerConfig{OutputTarget: "report.json"})
	require.NoError(t, err)
	require.IsType(t, &localFileHandler{}, handler)

	handler, err = NewOutputHandlers(nil, HandlerConfig{})
	require.NoError(t, err)
	require.IsType(t, &Reporter{}, handler)
}

func TestOutputHandlersStdoutCanRepeat(t *testing.T) {
	handler, err := NewOutputHandlers([]string{"compact", "json"}, HandlerConfig{})
	require.NoError(t, err)
	require.Len(t, handler, 2)
}

func TestOutputHandlersRejectSharedDestination(t *testing.T) {
	tests := []struct {
		name    string
		formats []string
		target  string
	}{
		{"same file=", []string{"json,file=out", "sarif,file=out"}, ""},
		{"file:// prefix is the same file", []string{"json,file=out", "sarif,file=file://out"}, ""},
		{"two outputs fall back to --output-target", []string{"json", "sarif"}, "out"},
		{"file= collides with --output-target", []string{"json", "sarif,file=out"}, "out"},
		{"target= collides with file=", []string{"json,target=out", "sarif,file=out"}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewOutputHandlers(tc.formats, HandlerConfig{OutputTarget: tc.target})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "both write to")
		})
	}
}

// TestOutputHandlersTargetVsFile pins the difference between the two options:
// target= is routed like --output-target, file= is always a local path.
func TestOutputHandlersTargetVsFile(t *testing.T) {
	const sqsURL = "https://sqs.us-east-1.amazonaws.com/123456789012/MyQueue"
	sbusURL := "https://my-sbus.servicebus.windows.net/my-queue"
	dir := t.TempDir()

	tests := []struct {
		name   string
		output string
		want   OutputHandler
	}{
		{"target= to SQS", "json,target=" + sqsURL, &awsSqsHandler{}},
		{"target= to Service Bus", "json,target=" + sbusURL, &azureSbusHandler{}},
		{"target= to a file", "json,target=" + filepath.Join(dir, "r.json"), &localFileHandler{}},
		{"target= to a directory", "hdf,target=" + dir, &hdfDirHandler{}},
		{"file= that reads like a queue URL", "json,file=" + sqsURL, &localFileHandler{}},
		{"file= to a directory", "hdf,file=" + dir, &hdfDirHandler{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler, err := NewOutputHandlers([]string{tc.output}, HandlerConfig{})
			require.NoError(t, err)
			require.IsType(t, tc.want, handler)
		})
	}
}
