// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"context"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/v10/policy"
)

type localFileHandler struct {
	file   string
	format Format
}

// we reuse the already implemented Reporter's WriteReport method by simply pointing the writer
// towards a file instead of stdout
func (h *localFileHandler) WriteReport(ctx context.Context, report *policy.ReportCollection) error {
	trimmedFile := strings.TrimPrefix(h.file, "file://")
	f, err := os.Create(trimmedFile)
	if err != nil {
		return err
	}
	defer f.Close() //nolint: errcheck
	reporter := NewReporter(h.format, false)
	reporter.out = f
	err = reporter.WriteReport(ctx, report)
	if err != nil {
		return err
	}
	log.Info().Str("file", trimmedFile).Msg("wrote report to file")
	return nil
}
