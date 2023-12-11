// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"testing"

	"github.com/stretchr/testify/require"
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
		rep, err := NewOutputHandler(ReportConfig{Format: "JSON", OutputTarget: sqsUrl})
		require.NoError(t, err, i)
		require.IsType(t, &awsSqsHandler{}, rep, i)
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

	for i, sqsUrl := range fileTargets {

		rep, err := NewOutputHandler(ReportConfig{Format: "JSON", OutputTarget: sqsUrl})
		require.NoError(t, err, i)
		require.IsType(t, &localFileHandler{}, rep, i)
	}
}

func TestCliReporter(t *testing.T) {
	rep, err := NewOutputHandler(ReportConfig{})
	require.NoError(t, err)
	require.IsType(t, &Reporter{}, rep)
}
