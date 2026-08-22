// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// The rules shared by every OCSF event class: how a cnspec score becomes a
// severity, a status and a compliance verdict, how timestamps are normalized, and
// what a finding carries besides its class-specific parts.

package reporter

import (
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"go.mondoo.com/cnspec/cli/reporter/ocsf"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/cli/printer"
)

func (c *ocsfConverter) findingInfo(query *policy.Mquery, score *policy.Score, title string) ocsf.FindingInfo {
	res := ocsf.FindingInfo{
		UID:           queryRuleID(query),
		Title:         title,
		Desc:          queryDescription(query),
		CreatedTime:   c.now,
		FirstSeenTime: ocsfMillis(score.GetFailureTime()),
		ModifiedTime:  ocsfMillis(score.GetValueModifiedTime()),
		Types:         []string{"Compliance Check"},
		DataSources:   []string{ocsfProductName},
	}
	if refs := queryRefs(query); len(refs) > 0 {
		res.SrcURL = refs[0].Url
	}
	return res
}

// checkUnmapped keeps the cnspec-specific values that OCSF has no attribute for.
func (c *ocsfConverter) checkUnmapped(query *policy.Mquery, score *policy.Score, ctx *ocsfAssetContext) map[string]string {
	res := map[string]string{}
	if ctx.assetMrn != "" {
		res["asset_mrn"] = ctx.assetMrn
	}
	if query.Mrn != "" {
		res["query_mrn"] = query.Mrn
	}
	if mql := strings.TrimSpace(queryMql(query)); mql != "" {
		res["mql"] = mql
	}
	if audit := queryAudit(query); audit != "" {
		res["audit"] = audit
	}
	if impact, ok := queryImpact(query); ok {
		res["impact"] = strconv.Itoa(int(impact))
	}
	if titles := ctx.policyTitles[query.Mrn]; len(titles) > 0 {
		res["policies"] = strings.Join(titles, ", ")
	}
	if score != nil {
		res["score_type"] = score.TypeLabel()
		res["completion"] = strconv.Itoa(int(score.Completion()))
		if score.Type == policy.ScoreType_Result {
			res["score"] = strconv.Itoa(int(score.Value))
			res["risk"] = strconv.Itoa(int(scoreRisk(score)))
		}
	}
	return res
}

// maxAssessmentBytes caps the "expected vs actual" detail of a check. A check
// that fails across thousands of resources can render megabytes of assessment,
// which bloats every row of the Parquet file and can exceed what an HTTP
// collector accepts in one event. The head of it is what makes a finding
// actionable; the rest is repetition.
const maxAssessmentBytes = 64 * 1024

// checkAssessment renders the "expected vs actual" detail of a check, which is
// what makes a failing finding actionable.
func checkAssessment(resolved *policy.ResolvedPolicy, report *policy.Report, query *policy.Mquery) string {
	if resolved == nil || resolved.ExecutionJob == nil {
		return ""
	}
	codeBundle := resolved.GetCodeBundle(query)
	if codeBundle == nil {
		return ""
	}
	assessment := policy.Query2Assessment(codeBundle, report)
	if assessment == nil {
		return ""
	}
	return truncateAssessment(strings.TrimSpace(printer.PlainNoColorPrinter.Assessment(codeBundle, assessment)))
}

func truncateAssessment(detail string) string {
	if len(detail) <= maxAssessmentBytes {
		return detail
	}
	// Cut on a rune boundary so the result stays valid UTF-8 and valid JSON.
	cut := maxAssessmentBytes
	for cut > 0 && !utf8.RuneStart(detail[cut]) {
		cut--
	}
	return detail[:cut] + "\n… assessment truncated"
}

func refURLs(query *policy.Mquery) []string {
	refs := queryRefs(query)
	res := make([]string, 0, len(refs))
	for _, ref := range refs {
		res = append(res, ref.Url)
	}
	return res
}

// ocsfCheckSeverity maps a check outcome to an OCSF severity. A check that passed
// or did not run is informational; a failing one carries the severity of its risk.
func ocsfCheckSeverity(score *policy.Score) int {
	if score == nil {
		return ocsf.SeverityInformational
	}
	switch score.Type {
	case policy.ScoreType_Error:
		// The check itself could not be evaluated. That is a gap in coverage, not
		// a security finding of its own.
		return ocsf.SeverityMedium
	case policy.ScoreType_Result:
		if score.Value == 100 {
			return ocsf.SeverityInformational
		}
		return ocsfSeverityFromRisk(scoreRisk(score))
	default:
		return ocsf.SeverityInformational
	}
}

// ocsfSeverityFromRisk maps a cnspec risk value (0-100) to an OCSF severity_id.
// The bands are the ones riskSeverityLabel uses, so the OCSF severity and the
// severity cnspec prints stay in sync.
func ocsfSeverityFromRisk(risk int32) int {
	switch {
	case risk >= 90:
		return ocsf.SeverityCritical
	case risk >= 70:
		return ocsf.SeverityHigh
	case risk >= 40:
		return ocsf.SeverityMedium
	case risk >= 1:
		return ocsf.SeverityLow
	default:
		return ocsf.SeverityInformational
	}
}

// ocsfFindingStatus maps a check outcome to the finding status and the label
// that goes with it. Findings cnspec produces are always newly observed; a
// skipped check is reported as suppressed, which is what "we deliberately did
// not evaluate this" means in OCSF, and a check that errored has no outcome.
//
// The label of an "Other" status is the cnspec outcome, not the word "Other":
// OCSF expects the string sibling of an Other enum to carry the producer's own
// value, and a validator flags one that just repeats the caption.
func ocsfFindingStatus(score *policy.Score) (int, string) {
	if score.GetType() == policy.ScoreType_Error {
		return ocsf.StatusOther, "Error"
	}
	switch scoreToSarifKind(score) {
	case "pass", "fail":
		return ocsf.StatusNew, ocsf.StatusName(ocsf.StatusNew)
	case "notApplicable":
		return ocsf.StatusSuppressed, ocsf.StatusName(ocsf.StatusSuppressed)
	case "informational":
		return ocsf.StatusOther, "Unscored"
	default:
		return ocsf.StatusUnknown, ocsf.StatusName(ocsf.StatusUnknown)
	}
}

// ocsfComplianceStatus maps a check outcome to the compliance verdict and its
// label. An errored check is deliberately not a Fail: nothing was evaluated, so
// calling it non-compliant would report an outage as a violation.
func ocsfComplianceStatus(score *policy.Score) (int, string) {
	if score.GetType() == policy.ScoreType_Error {
		return ocsf.ComplianceStatusUnknown, ocsf.ComplianceStatusName(ocsf.ComplianceStatusUnknown)
	}
	switch scoreToSarifKind(score) {
	case "pass":
		return ocsf.ComplianceStatusPass, ocsf.ComplianceStatusName(ocsf.ComplianceStatusPass)
	case "fail":
		return ocsf.ComplianceStatusFail, ocsf.ComplianceStatusName(ocsf.ComplianceStatusFail)
	case "notApplicable":
		return ocsf.ComplianceStatusOther, "Skipped"
	default:
		return ocsf.ComplianceStatusUnknown, ocsf.ComplianceStatusName(ocsf.ComplianceStatusUnknown)
	}
}

// ocsfMillis normalizes a cnspec timestamp to milliseconds. cnspec stores some of
// them in seconds and OCSF wants milliseconds throughout; anything below the
// year 5138 in seconds is far below a plausible millisecond timestamp, so the
// magnitude tells the two apart.
func ocsfMillis(ts int64) int64 {
	if ts <= 0 {
		return 0
	}
	if ts < 1e11 {
		return ts * 1000
	}
	return ts
}

// ocsfParseTime turns one of the date strings the vulnerability API returns into
// milliseconds since the epoch. Unparsable values are dropped rather than
// reported as the zero time, which would show up as the year 1 in the lake.
func ocsfParseTime(raw string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UnixMilli()
		}
	}
	return 0
}
