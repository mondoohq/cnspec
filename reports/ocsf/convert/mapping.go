// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// The rules shared by every OCSF event class: how a cnspec score becomes a
// severity, a status and a compliance verdict, how timestamps are normalized, and
// what a finding carries besides its class-specific parts.

package convert

import (
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/reports/ocsf"
	"go.mondoo.com/cnspec/reports/reportdoc"
	"go.mondoo.com/mql/cli/printer"
)

func (c *converter) findingInfo(query *policy.Mquery, score *policy.Score, title string) ocsf.FindingInfo {
	res := ocsf.FindingInfo{
		UID:           reportdoc.QueryRuleID(query),
		Title:         title,
		Desc:          reportdoc.QueryDescription(query),
		CreatedTime:   c.now,
		FirstSeenTime: millis(score.GetFailureTime()),
		ModifiedTime:  millis(score.GetValueModifiedTime()),
		Types:         []string{"Compliance Check"},
		DataSources:   []string{productName},
	}
	if refs := reportdoc.QueryRefs(query); len(refs) > 0 {
		res.SrcURL = refs[0].Url
	}
	return res
}

// checkUnmapped keeps the cnspec-specific values that OCSF has no attribute for.
func (c *converter) checkUnmapped(query *policy.Mquery, score *policy.Score, ctx *assetContext) map[string]string {
	res := map[string]string{}
	if ctx.assetMrn != "" {
		res["asset_mrn"] = ctx.assetMrn
	}
	if query.Mrn != "" {
		res["query_mrn"] = query.Mrn
	}
	if mql := strings.TrimSpace(reportdoc.QueryMql(query)); mql != "" {
		res["mql"] = mql
	}
	if audit := reportdoc.QueryAudit(query); audit != "" {
		res["audit"] = audit
	}
	if impact, ok := reportdoc.QueryImpact(query); ok {
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
			res["risk"] = strconv.Itoa(int(reportdoc.ScoreRisk(score)))
		}
	}
	return res
}

// maxUnmappedBytes caps any single rendered value carried in unmapped. A check
// that fails across thousands of resources renders megabytes of assessment, and a
// data query over a large resource set renders megabytes of JSON. Either bloats
// every row of the Parquet file and can exceed the per-event size an ingest
// pipeline accepts -- a Splunk HTTP Event Collector rejects an event over roughly
// 1 MB, so one uncapped value silently loses the whole event downstream, not just
// the value. The head of it is what makes the finding actionable; the rest is
// repetition.
const maxUnmappedBytes = 64 * 1024

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
	return truncateUnmapped(detail, "assessment")
}

// truncateUnmapped caps one unmapped value, naming in the marker what was cut so
// a consumer can tell a truncated value from a short one.
func truncateUnmapped(value, what string) string {
	if len(value) <= maxUnmappedBytes {
		return value
	}
	// Cut on a rune boundary so the result stays valid UTF-8 and valid JSON.
	cut := maxUnmappedBytes
	for cut > 0 && !utf8.RuneStart(value[cut]) {
		cut--
	}
	return value[:cut] + "\n… " + what + " truncated"
}

func refURLs(query *policy.Mquery) []string {
	refs := reportdoc.QueryRefs(query)
	res := make([]string, 0, len(refs))
	for _, ref := range refs {
		res = append(res, ref.Url)
	}
	return res
}

// checkSeverity maps a check outcome to an OCSF severity. A check that passed
// or did not run is informational; a failing one carries the severity of its risk.
func checkSeverity(score *policy.Score) int {
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
		return severityFromRisk(reportdoc.ScoreRisk(score))
	default:
		return ocsf.SeverityInformational
	}
}

// The three band mappers below all turn a reportdoc.Band into one OCSF enum, and
// they sit together because that is the only way to see that they agree on the
// boundaries and differ only in vocabulary. The boundaries themselves live in
// reportdoc.BandOf, so they cannot drift away from the severity cnspec prints;
// what varies here is that OCSF calls the lowest band Informational in one enum,
// Info in another and Unknown in a third.

// severityFromRisk maps a cnspec risk value (0-100) to an OCSF severity_id.
// OCSF calls the lowest band Informational, which is also what CVSS and Splunk
// consumers expect there.
func severityFromRisk(risk int32) int {
	switch reportdoc.BandOf(risk) {
	case reportdoc.BandCritical:
		return ocsf.SeverityCritical
	case reportdoc.BandHigh:
		return ocsf.SeverityHigh
	case reportdoc.BandMedium:
		return ocsf.SeverityMedium
	case reportdoc.BandLow:
		return ocsf.SeverityLow
	default:
		return ocsf.SeverityInformational
	}
}

// riskLevel maps a cnspec risk value to an OCSF risk_level_id. Same bands as the
// severity mapping, but risk_level is a separate enum with its own names, and it
// calls the lowest band Info.
func riskLevel(risk int32) int {
	switch reportdoc.BandOf(risk) {
	case reportdoc.BandCritical:
		return ocsf.RiskLevelCritical
	case reportdoc.BandHigh:
		return ocsf.RiskLevelHigh
	case reportdoc.BandMedium:
		return ocsf.RiskLevelMedium
	case reportdoc.BandLow:
		return ocsf.RiskLevelLow
	default:
		return ocsf.RiskLevelInfo
	}
}

// impactLevel maps a check's configured impact to an OCSF impact_id. Impact runs
// on the same 0-100 cnspec scale as risk, so it shares the boundaries; OCSF's
// impact enum has no informational band and calls its floor Unknown.
func impactLevel(impact int32) int {
	switch reportdoc.BandOf(impact) {
	case reportdoc.BandCritical:
		return ocsf.ImpactCritical
	case reportdoc.BandHigh:
		return ocsf.ImpactHigh
	case reportdoc.BandMedium:
		return ocsf.ImpactMedium
	case reportdoc.BandLow:
		return ocsf.ImpactLow
	default:
		return ocsf.ImpactUnknown
	}
}

// firstOrEmpty is the first item of a list, or "" for an empty one.
func firstOrEmpty(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return items[0]
}

// findingStatus maps a check outcome to the finding status and the label
// that goes with it. Findings cnspec produces are always newly observed; a
// skipped check is reported as suppressed, which is what "we deliberately did
// not evaluate this" means in OCSF, and a check that errored has no outcome.
//
// The label of an "Other" status is the cnspec outcome, not the word "Other":
// OCSF expects the string sibling of an Other enum to carry the producer's own
// value, and a validator flags one that just repeats the caption.
func findingStatus(score *policy.Score) (int, string) {
	switch reportdoc.OutcomeOf(score) {
	case reportdoc.OutcomePass, reportdoc.OutcomeFail:
		return ocsf.StatusNew, ocsf.StatusName(ocsf.StatusNew)
	case reportdoc.OutcomeError:
		return ocsf.StatusOther, "Error"
	case reportdoc.OutcomeSkipped:
		return ocsf.StatusSuppressed, ocsf.StatusName(ocsf.StatusSuppressed)
	case reportdoc.OutcomeUnscored:
		return ocsf.StatusOther, "Unscored"
	default:
		return ocsf.StatusUnknown, ocsf.StatusName(ocsf.StatusUnknown)
	}
}

// complianceStatus maps a check outcome to the compliance verdict and its
// label. An errored check is deliberately not a Fail: nothing was evaluated, so
// calling it non-compliant would report an outage as a violation.
func complianceStatus(score *policy.Score) (int, string) {
	switch reportdoc.OutcomeOf(score) {
	case reportdoc.OutcomePass:
		return ocsf.ComplianceStatusPass, ocsf.ComplianceStatusName(ocsf.ComplianceStatusPass)
	case reportdoc.OutcomeFail:
		return ocsf.ComplianceStatusFail, ocsf.ComplianceStatusName(ocsf.ComplianceStatusFail)
	case reportdoc.OutcomeSkipped:
		return ocsf.ComplianceStatusOther, "Skipped"
	default:
		return ocsf.ComplianceStatusUnknown, ocsf.ComplianceStatusName(ocsf.ComplianceStatusUnknown)
	}
}

// millis normalizes a cnspec timestamp to milliseconds. cnspec stores some of
// them in seconds and OCSF wants milliseconds throughout; anything below the
// year 5138 in seconds is far below a plausible millisecond timestamp, so the
// magnitude tells the two apart.
func millis(ts int64) int64 {
	if ts <= 0 {
		return 0
	}
	if ts < 1e11 {
		return ts * 1000
	}
	return ts
}

// parseTime turns one of the date strings the vulnerability API returns into
// milliseconds since the epoch. Unparsable values are dropped rather than
// reported as the zero time, which would show up as the year 1 in the lake.
func parseTime(raw string) int64 {
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
