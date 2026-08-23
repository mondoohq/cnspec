// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/internal/reportfixture"
)

// What the document says is the hdf package's business; these cover the wiring that
// only exists here - that `-o hdf` reaches the converter at all, and which handler a
// given --output-target picks.

func TestHDFReporterFormat(t *testing.T) {
	conf, err := ParseConfig("hdf")
	require.NoError(t, err)
	assert.Equal(t, FormatHDF, conf.format)

	buf := bytes.Buffer{}
	require.NoError(t, NewReporter(conf, false).WithOutput(&buf).
		WriteReport(t.Context(), reportfixture.Sample()))

	var report struct {
		Profiles []json.RawMessage `json:"profiles"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &report))
	assert.NotEmpty(t, report.Profiles)
}

// TestOutputHandlerHDFDirectory covers the one output target whose handler depends
// on the format: an OHDF document describes a single asset, so a directory target
// means "a file per asset" rather than "a file called that".
func TestOutputHandlerHDFDirectory(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name   string
		format string
		target string
		want   OutputHandler
	}{
		{"hdf into an existing directory", "hdf", dir, &hdfDirHandler{}},
		{"hdf into a path asking for one", "hdf", filepath.Join(dir, "out") + "/", &hdfDirHandler{}},
		{"hdf into a file", "hdf", filepath.Join(dir, "report.json"), &localFileHandler{}},
		{"another format into a directory", "json", dir, &localFileHandler{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rep, err := NewOutputHandler(HandlerConfig{Format: test.format, OutputTarget: test.target})
			require.NoError(t, err)
			require.IsType(t, test.want, rep)
		})
	}
}
