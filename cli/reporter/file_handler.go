// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"context"
	"os"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/v9/policy"
	_ "gocloud.dev/pubsub/awssnssqs"
)

type localFileHandler struct {
	file   string
	format Format
}

// we reuse the already implemented Reporter's WriteReport method by simply pointing the writer
// towards a file instead of stdout
func (h *localFileHandler) WriteReport(ctx context.Context, report *policy.ReportCollection) error {
	f, err := os.Create(h.file)
	if err != nil {
		return err
	}
	defer f.Close() //nolint: errcheck
	reporter := NewReporter(h.format, false)
	if err != nil {
		return err
	}
	reporter.out = f
	err = reporter.WriteReport(ctx, report)
	if err != nil {
		return err
	}
	log.Info().Str("file", h.file).Msg("wrote report to file")
	return nil
}
