// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOutputHandlerAwsSqs(t *testing.T) {
	validSqsUrls := []string{
		"https://sqs.us-east-1.amazonaws.com/123456789012/MyQueue",
		"http://sqs.us-east-1.amazonaws.com/123456789012/MyQueue",
		"https://sqs.eu-central-1.amazonaws.com/123456789012/MyQueue",
		"http://sqs.eu-central-1.amazonaws.com/123456789012/MyQueue",
		"sqs.us-east-1.amazonaws.com/123456789012/MyQueue",
		"sqs.eu-central-1.amazonaws.com/123456789012/MyQueue",
	}

	for i, sqsUrl := range validSqsUrls {
		rep, err := NewOutputHandler(sqsUrl, "JSON")
		require.NoError(t, err, i)
		require.IsType(t, &awsSqsHandler{}, rep, i)
	}

	invalidSqsUrls := []string{
		"https://sqss.us-east-1.amazonaws.com/123456789012/MyQueue",
		"http://sqss.us-east-1.amazonaws.com/123456789012/MyQueue",
		"sqss.us-east-1.amazonaws.com/123456789012/MyQueue",
		"sqss.eu-central-1.amazonaws.com/123456789012/MyQueue",
		"sqs.europe-central-1.amazonaws.com/123456789012/MyQueue",
		"somethingtotallyrandom",
	}

	for i, sqsUrl := range invalidSqsUrls {
		_, err := NewOutputHandler(sqsUrl, "JSON")
		require.Error(t, err, i)
	}
}

func TestOutputHandlerLocal(t *testing.T) {
	validFileTarget := []string{
		"file:///root/test",
		"file:///root/test.json",
		"file://root/test.json",
	}

	for i, sqsUrl := range validFileTarget {
		rep, err := NewOutputHandler(sqsUrl, "JSON")
		require.NoError(t, err, i)
		require.IsType(t, &localFileHandler{}, rep, i)
	}

	invalidFileTargets := []string{
		"filee:///root/test",
		"/root/json",
	}

	for i, sqsUrl := range invalidFileTargets {
		_, err := NewOutputHandler(sqsUrl, "JSON")
		require.Error(t, err, i)
	}
}

func TestNoOpReporter(t *testing.T) {
	rep, err := NewOutputHandler("", "JSON")
	require.NoError(t, err)
	require.IsType(t, &noopHandler{}, rep)
}
