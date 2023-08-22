// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package components

type ReportFindingRow struct {
	Score     int32    `json:"score"`
	Name      string   `json:"package"`
	Installed string   `json:"installed"`
	Fixed     string   `json:"vulnerable"`
	Available string   `json:"available"`
	Advisory  string   `json:"advisory"`
	Cves      []string `json:"cves"`
}
