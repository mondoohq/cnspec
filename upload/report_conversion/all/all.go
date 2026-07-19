// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package all blank-imports every cnspec-resident converter so their init()
// registrations run. Import it (with _) from the CLI and any server entry point
// that needs the full set of standard-format converters.
package all

import (
	_ "go.mondoo.com/cnspec/v13/upload/report_conversion/defectdojo"
	_ "go.mondoo.com/cnspec/v13/upload/report_conversion/sarif"
)
