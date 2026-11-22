// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import "go.mondoo.com/cnspec/policy"

// vulnNotAssessedReason says why an asset has no vulnerability results. Every
// format that reports vulnerabilities uses it, so that missing data reads the
// same everywhere and never as zero findings.
func vulnNotAssessedReason(a *policy.VulnAssessment) string {
	msg := a.GetMessage()
	switch a.GetState() {
	case policy.VulnResultState_VULN_RESULT_STATE_PENDING:
		if msg == "" {
			msg = "Mondoo Platform did not finish analyzing vulnerabilities in time"
		}
		return msg
	case policy.VulnResultState_VULN_RESULT_STATE_NOT_ASSESSED:
		if msg == "" {
			msg = "vulnerabilities are not analyzed for this platform"
		}
		return msg
	case policy.VulnResultState_VULN_RESULT_STATE_FAILED:
		if msg == "" {
			return "could not get vulnerability results"
		}
		return "could not get vulnerability results: " + msg
	case policy.VulnResultState_VULN_RESULT_STATE_READY:
		return "the vulnerability report has no statistics"
	}
	return "no vulnerability results"
}
