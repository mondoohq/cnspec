// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/muesli/termenv"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/cli/theme"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream/mvd"
)

type defaultVulnReporter struct {
	*Reporter
	isCompact bool
	isSummary bool
	target    string
	out       io.Writer
	data      *mvd.VulnReport
}

func (r *defaultVulnReporter) print() error {
	// catch case where the scan was not successful and no bundle was fetched from server
	if r.data == nil {
		log.Debug().Msg("report does not contain any data")
		return nil
	}

	header := fmt.Sprintf("\nTarget:     %s\n", r.target)
	r.out.Write([]byte(termenv.String(header).Foreground(theme.DefaultTheme.Colors.Primary).String()))
	summaryDivider := strings.Repeat("=", utf8.RuneCountInString(header))
	r.out.Write([]byte(termenv.String(summaryDivider + "\n\n").Foreground(theme.DefaultTheme.Colors.Secondary).String()))
	r.out.Write([]byte(RenderVulnerabilityStats(r.data)))
	if !r.isSummary {
		r.out.Write([]byte(RenderVulnReportDetailed(r.data, !r.isCompact)))
	}
	return nil
}
