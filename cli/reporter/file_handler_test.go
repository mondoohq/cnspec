// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v9/policy"
)

func TestFileHandler(t *testing.T) {
	reportCollectionRaw, err := os.ReadFile("./testdata/report-ubuntu.json")
	require.NoError(t, err)

	yr := &policy.ReportCollection{}
	err = json.Unmarshal(reportCollectionRaw, yr)
	require.NoError(t, err)

	now := time.Now().Format(time.RFC3339)
	fileName := fmt.Sprintf("/tmp/%s-testfilehandler.json", now)
	config := ReportConfig{Format: "compact", OutputTarget: fileName}
	handler, err := NewOutputHandler(config)
	require.NoError(t, err)
	err = handler.WriteReport(context.Background(), yr)
	require.NoError(t, err)
	data, err := os.ReadFile(fileName)
	require.NoError(t, err)

	strData := string(data)
	assert.Contains(t, strData, "✕ Fail:         Ensure")
	assert.Contains(t, strData, ". Skipped:      Set")
	assert.Contains(t, strData, "! Error:        Set")
	assert.Contains(t, strData, "✓ Pass:  A 100  Ensure")
	assert.Contains(t, strData, "✕ Fail:  F   0  Ensure")
	err = os.Remove(fileName)
	require.NoError(t, err)
}
