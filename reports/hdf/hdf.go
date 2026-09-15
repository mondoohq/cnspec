// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package hdf

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec"
	"go.mondoo.com/cnspec/internal/reportfile"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/reports/reportdoc"
	"go.mondoo.com/mql/cli/printer"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/mql/utils/iox"
)

const (
	hdfToolName          = "cnspec"
	hdfAssetErrorID      = "asset-error"
	hdfVulnPackageID     = "vulnerable-package"
	hdfVulnGroupID       = "cnspec-vulnerabilities"
	hdfProfileStatus     = "loaded"
	hdfNistTagKey        = "compliance/nist-sp-800-53-rev5"
	hdfNistTagPrefix     = "nist-sp-800-53-rev5-"
	hdfNistUnmappedTag   = "UM-1"
	hdfSourceLocationRef = "cnspec"
)

// OHDF result statuses. Heimdall derives the control's overall status from these
// together with the impact - see hdfControlImpact.
const (
	hdfStatusPassed  = "passed"
	hdfStatusFailed  = "failed"
	hdfStatusSkipped = "skipped"
	hdfStatusError   = "error"
)

// hdfMinImpact is the impact floor for a check that carries no severity. Two values
// below it would lose the finding: an impact of exactly 0 means "Not Applicable" in
// Heimdall, which has to stay reserved for checks the scan deliberately excluded,
// and anything under 0.1 lands in the "none" severity band, which the SAF CLI's
// summary buckets have no counter for - an unrated check that errored would drop out
// of the tally entirely. 0.1 is the lowest impact the toolchain actually counts.
const hdfMinImpact = 0.1

// hdfTimeNow is the clock used for result timestamps. It is a variable so tests can
// pin it and compare rendered documents byte for byte.
var hdfTimeNow = time.Now

type hdfReport struct {
	Platform    hdfPlatform     `json:"platform"`
	Version     string          `json:"version"`
	Statistics  hdfStatistics   `json:"statistics"`
	Profiles    []*hdfProfile   `json:"profiles"`
	Passthrough *hdfPassthrough `json:"passthrough,omitempty"`
}

type hdfPlatform struct {
	Name     string `json:"name"`
	Release  string `json:"release"`
	TargetID string `json:"target_id"`
}

type hdfStatistics struct {
	Duration *float64 `json:"duration"`
}

type hdfProfile struct {
	Name           string              `json:"name"`
	Title          *string             `json:"title"`
	Version        *string             `json:"version"`
	Maintainer     *string             `json:"maintainer"`
	Summary        *string             `json:"summary"`
	License        *string             `json:"license"`
	Copyright      *string             `json:"copyright"`
	CopyrightEmail *string             `json:"copyright_email"`
	Supports       []map[string]string `json:"supports"`
	Attributes     []any               `json:"attributes"`
	Depends        []any               `json:"depends"`
	Groups         []*hdfGroup         `json:"groups"`
	Sha256         string              `json:"sha256"`
	Status         string              `json:"status"`
	Controls       []*hdfControl       `json:"controls"`
}

type hdfGroup struct {
	ID       string   `json:"id"`
	Title    *string  `json:"title"`
	Controls []string `json:"controls"`
}

type hdfControl struct {
	ID             string            `json:"id"`
	Title          *string           `json:"title"`
	Desc           *string           `json:"desc"`
	Descriptions   []hdfDescription  `json:"descriptions"`
	Impact         float64           `json:"impact"`
	Refs           []hdfRef          `json:"refs"`
	Tags           map[string]any    `json:"tags"`
	Code           string            `json:"code"`
	SourceLocation hdfSourceLocation `json:"source_location"`
	Results        []hdfResult       `json:"results"`
}

type hdfDescription struct {
	Label string `json:"label"`
	Data  string `json:"data"`
}

type hdfRef struct {
	Ref string `json:"ref,omitempty"`
	URL string `json:"url,omitempty"`
}

type hdfSourceLocation struct {
	Ref  string `json:"ref"`
	Line int    `json:"line"`
}

type hdfResult struct {
	Status      string   `json:"status"`
	CodeDesc    string   `json:"code_desc"`
	Message     string   `json:"message,omitempty"`
	SkipMessage string   `json:"skip_message,omitempty"`
	Resource    string   `json:"resource,omitempty"`
	RunTime     *float64 `json:"run_time,omitempty"`
	StartTime   string   `json:"start_time"`
}

type hdfPassthrough struct {
	AuxiliaryData []hdfAuxiliaryData `json:"auxiliary_data"`
}

type hdfAuxiliaryData struct {
	Name string `json:"name"`
	Data any    `json:"data"`
}

// hdfAssetContext carries everything the converter needs about the asset whose
// profile is currently being built.
type hdfAssetContext struct {
	assetMrn     string
	asset        *inventory.Asset
	platformKeys map[string]bool
	policyTitles map[string][]string
	startTime    string
}

// hdfDocument is one rendered OHDF document together with the asset it covers, so
// callers that write a file per asset can name the file after it.
type hdfDocument struct {
	assetMrn string
	name     string
	report   *hdfReport
}

// Convert converts a ReportCollection into OHDF (Heimdall Data Format) and
// writes it to a single stream.
//
// A scan of one asset - the usual case in CI - produces exactly one document. A
// scan that covered several cannot: an OHDF document describes a single target, and
// Heimdall and the SAF CLI tally only the first profile in a document, so folding
// several assets into one would silently drop every finding but the first asset's.
// Those are written as a JSON array of documents instead, and the caller is pointed
// at ConvertToDir, which writes each one to its own file.
func Convert(r *policy.ReportCollection, out iox.OutputHelper) error {
	conv, err := newHDFConverter(r)
	if err != nil {
		return err
	}

	switch len(conv.assetMrns) {
	case 0:
		return writeHDFValue(hdfEmptyReport(), out)
	case 1:
		return writeHDFValue(conv.document(conv.assetMrns[0]).report, out)
	}

	log.Warn().Int("assets", len(conv.assetMrns)).
		Msg("an OHDF document describes a single asset, so this scan is written as a JSON array of documents. " +
			"Pass a directory to --output-target to get one OHDF file per asset instead")

	// Rendered and written one at a time: a scan of a large fleet would otherwise
	// hold every converted document in memory at once, on top of the report
	// collection it was built from.
	if err := out.WriteString("["); err != nil {
		return err
	}
	for i, assetMrn := range conv.assetMrns {
		if i > 0 {
			if err := out.WriteString(","); err != nil {
				return err
			}
		}
		data, err := json.Marshal(conv.document(assetMrn).report)
		if err != nil {
			return err
		}
		if _, err := out.Write(data); err != nil {
			return err
		}
	}
	return out.WriteString("]\n")
}

// ConvertToDir writes one OHDF document per scanned asset into dir, named after
// the asset. This is the form the MITRE tooling expects for a multi-target scan.
func ConvertToDir(r *policy.ReportCollection, dir string) ([]string, error) {
	conv, err := newHDFConverter(r)
	if err != nil {
		return nil, err
	}

	// 0700, not 0755: an OHDF document carries the MQL source of every check, the
	// rendered assessment with the observed values in it, and the asset's platform
	// and cloud identity. A scan is routinely run as root, and a world-readable
	// directory under it hands all of that to every local account.
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}

	// Asset names are free-form, so two of them can sanitize to the same filename.
	used := map[string]int{}
	written := make([]string, 0, len(conv.assetMrns))
	var errs []error
	for _, assetMrn := range conv.assetMrns {
		// Built inside the loop so only one document is live at a time.
		doc := conv.document(assetMrn)
		path := filepath.Join(dir, hdfUniqueName(hdfFileBase(doc), used)+".hdf.json")
		if err := writeHDFFile(doc.report, path); err != nil {
			// Keep going: one asset the filesystem rejects should not cost the
			// report of every asset after it. The errors are returned together.
			errs = append(errs, fmt.Errorf("write report for %q: %w", doc.name, err))
			continue
		}
		written = append(written, path)
	}
	return written, errors.Join(errs...)
}

// writeHDFFile renders one document to its own file.
func writeHDFFile(report *hdfReport, path string) error {
	// Not os.Create: it would leave the document 0644, and it follows a symlink.
	// The filename is derived from the asset name, so anyone who can write the
	// output directory and knows what is being scanned can pre-place a link at it
	// and have a scan -- frequently running as root -- truncate and overwrite the
	// link's target with the report. See internal/reportfile and, for what is in
	// the document, ConvertToDir.
	f, err := reportfile.Create(path)
	if err != nil {
		return err
	}

	writer := iox.IOWriter{Writer: f}
	if err := writeHDFValue(report, &writer); err != nil {
		f.Close() //nolint:errcheck // the write error is the one worth reporting
		return err
	}
	return f.Close()
}

// hdfConverter holds the state every document of a scan shares - the query index,
// the policy titles, the asset order - so it is derived once rather than per asset.
type hdfConverter struct {
	r            *policy.ReportCollection
	bundle       *policy.PolicyBundleMap
	queries      map[string]*policy.Mquery
	policyTitles map[string][]string
	assetMrns    []string
}

func newHDFConverter(r *policy.ReportCollection) (*hdfConverter, error) {
	if r == nil {
		return &hdfConverter{}, nil
	}
	if r.Bundle == nil {
		return nil, fmt.Errorf("no policy bundle found")
	}

	bundle := r.Bundle.ToMap()
	return &hdfConverter{
		r:            r,
		bundle:       bundle,
		queries:      reportdoc.QueryMap(bundle),
		policyTitles: reportdoc.PolicyTitlesByQuery(bundle),
		assetMrns:    slices.Sorted(maps.Keys(r.Assets)),
	}, nil
}

// document renders the OHDF document for one asset.
func (c *hdfConverter) document(assetMrn string) *hdfDocument {
	assetObj := c.r.Assets[assetMrn]
	if assetObj == nil {
		assetObj = &inventory.Asset{Name: assetMrn}
	}

	ctx := &hdfAssetContext{
		assetMrn:     assetMrn,
		asset:        assetObj,
		platformKeys: reportdoc.PlatformRemediationKeys(assetObj.Platform),
		policyTitles: c.policyTitles,
		startTime:    hdfStartTime(c.r.Reports[assetMrn]),
	}

	report := hdfEmptyReport()
	report.Platform = hdfPlatformFor(assetObj, assetMrn)
	report.Statistics = hdfStatisticsFor(c.r.Reports[assetMrn])
	report.Profiles = []*hdfProfile{hdfAssetProfile(c.r, ctx, c.bundle, c.queries)}
	report.Passthrough = hdfPassthroughFor(c.r, ctx)

	return &hdfDocument{
		assetMrn: assetMrn,
		name:     hdfProfileName(assetObj, assetMrn),
		report:   report,
	}
}

// hdfDocuments renders every document of a scan at once. Only the tests use it; the
// writers render one at a time so a large fleet does not have to fit in memory.
func hdfDocuments(r *policy.ReportCollection) ([]*hdfDocument, error) {
	conv, err := newHDFConverter(r)
	if err != nil {
		return nil, err
	}

	docs := make([]*hdfDocument, 0, len(conv.assetMrns))
	for _, assetMrn := range conv.assetMrns {
		docs = append(docs, conv.document(assetMrn))
	}
	return docs, nil
}

// hdfEmptyReport is a well-formed OHDF document with no findings in it.
func hdfEmptyReport() *hdfReport {
	return &hdfReport{
		Platform: hdfPlatform{Name: hdfToolName, Release: cnspec.GetVersion()},
		Version:  cnspec.GetVersion(),
		Profiles: []*hdfProfile{},
	}
}

// hdfMaxFileBase caps the stem of a written filename. Asset names are free-form and
// routinely long - a full ARN, an image reference with a digest, a namespaced
// Kubernetes object - while most filesystems stop at 255 bytes for one path element.
// The cap leaves room for the ".hdf.json" suffix and the "-N" a collision adds.
const hdfMaxFileBase = 200

// hdfFileBase turns an asset into a filename stem. Asset names carry spaces, slashes
// and colons (an image reference, a cloud resource id), none of which belong in a
// path, and an asset with no usable name at all falls back to its MRN digest.
func hdfFileBase(doc *hdfDocument) string {
	var b strings.Builder
	for _, r := range doc.name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}

	name := strings.Trim(b.String(), "-.")
	if name == "" {
		return "asset-" + hdfSha256(doc.assetMrn)[:12]
	}

	// Truncating alone would collide two assets that share a long prefix - think a
	// registry path where only the trailing digest differs - so the MRN digest goes
	// on the end to keep them apart.
	if len(name) > hdfMaxFileBase {
		digest := hdfSha256(doc.assetMrn)[:12]
		name = strings.Trim(name[:hdfMaxFileBase-len(digest)-1], "-.") + "-" + digest
	}
	return name
}

// hdfUniqueName suffixes a name that has already been used, so no document
// overwrites another.
func hdfUniqueName(name string, used map[string]int) string {
	used[name]++
	if n := used[name]; n > 1 {
		return name + "-" + strconv.Itoa(n)
	}
	return name
}

// hdfPlatformFor describes the target of a document: the asset it covers.
func hdfPlatformFor(assetObj *inventory.Asset, assetMrn string) hdfPlatform {
	res := hdfPlatform{Name: hdfToolName, Release: cnspec.GetVersion()}
	if assetObj.Platform != nil && assetObj.Platform.Name != "" {
		res.Name = assetObj.Platform.Name
		res.Release = assetObj.Platform.Version
	}
	res.TargetID = hdfTargetID(assetObj, assetMrn)
	return res
}

// hdfTargetID identifies the asset. The platform id is the stable, tool-independent
// handle; the asset MRN is the fallback for assets that have none.
func hdfTargetID(assetObj *inventory.Asset, assetMrn string) string {
	if len(assetObj.PlatformIds) > 0 && assetObj.PlatformIds[0] != "" {
		return assetObj.PlatformIds[0]
	}
	return assetMrn
}

// hdfStatisticsFor reports how long the asset took to scan, when its report carries
// the timestamps to derive it from. It stays null otherwise - OHDF consumers treat a
// missing duration as "not reported", but would read a 0 as an instant scan.
func hdfStatisticsFor(report *policy.Report) hdfStatistics {
	if report == nil || report.Created <= 0 || report.Modified <= report.Created {
		return hdfStatistics{}
	}
	duration := float64(report.Modified - report.Created)
	return hdfStatistics{Duration: &duration}
}

// hdfStartTime is the time the checks of an asset ran, formatted the way OHDF
// expects. It falls back to the current time for reports that carry no timestamp
// (local scans do not set one).
func hdfStartTime(report *policy.Report) string {
	if report != nil && report.Created > 0 {
		return time.Unix(report.Created, 0).UTC().Format(time.RFC3339)
	}
	return hdfTimeNow().UTC().Format(time.RFC3339)
}

// hdfAssetProfile builds the one profile of an asset's document: its check results,
// its scan error if it had one, and its vulnerability findings.
//
// It has to be a single profile. Heimdall and the SAF CLI resolve a document down to
// one root profile and tally only that one, so anything in a second, unlinked
// profile is dropped without a word - a vulnerability finding parked in its own
// profile never reaches the summary. Grouping keeps the two kinds distinguishable.
func hdfAssetProfile(r *policy.ReportCollection, ctx *hdfAssetContext, bundle *policy.PolicyBundleMap, queries map[string]*policy.Mquery) *hdfProfile {
	profile := hdfNewProfile(hdfProfileName(ctx.asset, ctx.assetMrn))
	profile.Title = strPtr("cnspec policy scan of " + hdfAssetLabel(ctx.asset, ctx.assetMrn))
	profile.Summary = strPtr(hdfAssetSummary(ctx.asset))
	profile.Supports = hdfSupports(ctx.asset.Platform)

	// controlIDs maps a check MRN to the control it produced, so the policy groups
	// below can reference only the checks that actually reported for this asset.
	controlIDs := map[string]string{}

	report := r.Reports[ctx.assetMrn]
	resolved := r.ResolvedPolicies[ctx.assetMrn]
	if report != nil && resolved != nil && resolved.CollectorJob != nil {
		for _, id := range slices.Sorted(maps.Keys(report.Scores)) {
			if _, ok := resolved.CollectorJob.ReportingQueries[id]; !ok {
				continue
			}
			query, ok := queries[id]
			if !ok {
				continue
			}

			control := hdfCheckControl(ctx, resolved, report, query, report.Scores[id])
			profile.Controls = append(profile.Controls, control)
			if query.Mrn != "" {
				controlIDs[query.Mrn] = control.ID
			}
		}
	}

	if errMsg, ok := r.Errors[ctx.assetMrn]; ok {
		profile.Controls = append(profile.Controls, hdfAssetErrorControl(ctx, errMsg))
	}

	profile.Groups = hdfPolicyGroups(bundle, controlIDs)

	if vulnControls := hdfVulnControls(r.VulnReports[ctx.assetMrn], ctx); len(vulnControls) > 0 {
		ids := make([]string, 0, len(vulnControls))
		for _, control := range vulnControls {
			ids = append(ids, control.ID)
		}
		sort.Strings(ids)
		profile.Controls = append(profile.Controls, vulnControls...)
		profile.Groups = append(profile.Groups, &hdfGroup{
			ID:       hdfVulnGroupID,
			Title:    strPtr("Vulnerabilities"),
			Controls: ids,
		})
	}

	profile.Sha256 = hdfProfileChecksum(resolved, ctx.assetMrn, profile.Controls)
	return profile
}

// hdfCheckControl turns a check and its score into an OHDF control.
func hdfCheckControl(ctx *hdfAssetContext, resolved *policy.ResolvedPolicy, report *policy.Report, query *policy.Mquery, score *policy.Score) *hdfControl {
	control := &hdfControl{
		ID:             reportdoc.QueryRuleID(query),
		Impact:         hdfControlImpact(query, score),
		Descriptions:   []hdfDescription{},
		Refs:           hdfRefs(reportdoc.QueryRefs(query)),
		Code:           reportdoc.QueryMql(query),
		SourceLocation: hdfSourceLocation{Ref: hdfSourceLocationRef, Line: 1},
	}

	if query.Title != "" {
		control.Title = strPtr(query.Title)
	}
	if desc := strings.TrimSpace(reportdoc.QueryDescription(query)); desc != "" {
		control.Desc = strPtr(desc)
	}
	if audit := reportdoc.QueryAudit(query); audit != "" {
		control.Descriptions = append(control.Descriptions, hdfDescription{Label: "check", Data: audit})
	}
	if rem := reportdoc.QueryRemediation(query, ctx.platformKeys); rem != "" {
		control.Descriptions = append(control.Descriptions, hdfDescription{Label: "fix", Data: rem})
	}

	control.Tags = hdfCheckTags(ctx, query, control.Impact)
	control.Results = hdfCheckResults(ctx, resolved, report, query, score)
	return control
}

// hdfControlImpact maps a check's cnspec impact (0-100) onto the OHDF 0.0-1.0
// scale. The severity bands line up exactly - cnspec calls 70 HIGH and Heimdall
// calls 0.7 high - so the conversion is a straight division.
//
// Two values are special: an impact of 0 renders as "Not Applicable" in Heimdall,
// which is the right reading for a check the scan deliberately excluded and the
// wrong one for every other check, so those are floored at hdfMinImpact.
func hdfControlImpact(query *policy.Mquery, score *policy.Score) float64 {
	if score != nil {
		switch score.Type {
		case policy.ScoreType_Skip, policy.ScoreType_OutOfScope, policy.ScoreType_Disabled:
			return 0
		}
	}

	risk := hdfControlRisk(query, score)
	if risk >= 100 {
		return 1
	}
	if risk <= 0 {
		return hdfMinImpact
	}
	return float64(risk) / 100
}

// hdfControlRisk is the cnspec risk (0-100) a control is reported at: its configured
// impact, or - for a check that has none - what its score implies. The fallback is
// limited to scored results on purpose: an errored or skipped check carries a score
// value of 0, and reading that as risk 100 would report every one of them as
// critical.
func hdfControlRisk(query *policy.Mquery, score *policy.Score) int32 {
	if risk, ok := reportdoc.QueryImpact(query); ok {
		return risk
	}
	if score != nil && score.Type == policy.ScoreType_Result {
		return reportdoc.ScoreRisk(score)
	}
	return 0
}

// hdfCheckTags carries the check's metadata. `nist` and `severity` are the keys
// Heimdall reads for its compliance and severity views; the rest is cnspec context
// that would otherwise be lost.
func hdfCheckTags(ctx *hdfAssetContext, query *policy.Mquery, impact float64) map[string]any {
	tags := map[string]any{
		"nist":     hdfNistTags(query),
		"severity": hdfSeverityLabel(impact),
		"asset":    ctx.asset.Name,
	}
	if ctx.assetMrn != "" {
		tags["assetMrn"] = ctx.assetMrn
	}
	if query.Mrn != "" {
		tags["queryMrn"] = query.Mrn
	}
	if mql := strings.TrimSpace(reportdoc.QueryMql(query)); mql != "" {
		tags["mql"] = mql
	}
	if titles := ctx.policyTitles[query.Mrn]; len(titles) > 0 {
		tags["policies"] = titles
	}
	for key, value := range reportdoc.QueryComplianceTags(query) {
		tags[key] = value
	}
	return tags
}

// hdfSeverityLabel is the severity band an impact falls into, using the thresholds
// Heimdall and the SAF CLI apply. It is derived from the impact rather than from the
// cnspec risk so the two can never disagree: the SAF summary buckets a finding by
// this tag, and a control tagged "none" is dropped from the tally no matter what its
// impact says.
func hdfSeverityLabel(impact float64) string {
	switch {
	case impact >= 0.9:
		return "critical"
	case impact >= 0.7:
		return "high"
	case impact >= 0.4:
		return "medium"
	case impact >= 0.1:
		return "low"
	default:
		return "none"
	}
}

var hdfNistControlRe = regexp.MustCompile(`^([a-z]{2})-(\d+)(?:-(\d+))?$`)

// hdfNistTags renders a check's NIST SP 800-53 mapping the way Heimdall's
// compliance views expect it: "nist-sp-800-53-rev5-si-7" becomes "SI-7", and a
// control enhancement like "ac-2-1" becomes "AC-2 (1)". Checks with no mapping get
// MITRE's "UM-1" marker for unmapped controls, so they still appear in the views.
func hdfNistTags(query *policy.Mquery) []string {
	raw := reportdoc.QueryComplianceTags(query)[hdfNistTagKey]
	match := hdfNistControlRe.FindStringSubmatch(strings.TrimPrefix(raw, hdfNistTagPrefix))
	if match == nil {
		return []string{hdfNistUnmappedTag}
	}

	control := strings.ToUpper(match[1]) + "-" + match[2]
	if match[3] != "" {
		control += " (" + match[3] + ")"
	}
	return []string{control}
}

// hdfCheckResults renders the outcome of a check. A check that failed on resources
// carrying source context (Terraform and friends) reports one result per failing
// resource so consumers can point at the offending file; everything else reports a
// single result.
func hdfCheckResults(ctx *hdfAssetContext, resolved *policy.ResolvedPolicy, report *policy.Report, query *policy.Mquery, score *policy.Score) []hdfResult {
	status := hdfStatus(score)

	codeDesc := strings.TrimSpace(reportdoc.QueryMql(query))
	if codeDesc == "" {
		codeDesc = query.Title
	}
	if codeDesc == "" {
		codeDesc = reportdoc.QueryRuleID(query)
	}

	base := hdfResult{
		Status:    status,
		CodeDesc:  codeDesc,
		StartTime: ctx.startTime,
	}

	switch status {
	case hdfStatusSkipped:
		base.SkipMessage = hdfSkipMessage(score)
	case hdfStatusError:
		base.Message = score.MessageLine()
	}

	// A passing check has nothing to explain and no failing resource to point at,
	// so the assessment is never read for one. Building and rendering it anyway
	// costs the majority of the work in a healthy scan.
	if status == hdfStatusPassed {
		return []hdfResult{base}
	}

	var assessment *llx.Assessment
	var codeBundle *llx.CodeBundle
	if report != nil && resolved != nil && resolved.ExecutionJob != nil {
		if codeBundle = resolved.GetCodeBundle(query); codeBundle != nil {
			assessment = policy.Query2Assessment(codeBundle, report)
		}
	}

	var detail string
	if assessment != nil && codeBundle != nil {
		detail = strings.TrimSpace(printer.PlainNoColorPrinter.Assessment(codeBundle, assessment))
	}

	var locations []llx.SourceContext
	if assessment != nil && codeBundle != nil {
		for _, sc := range codeBundle.FailingResourceContexts(assessment) {
			if sc.Path != "" {
				locations = append(locations, sc)
			}
		}
	}

	if len(locations) == 0 {
		if status == hdfStatusFailed && detail != "" {
			base.Message = detail
		}
		if base.Message == "" && status != hdfStatusPassed {
			base.Message = score.MessageLine()
		}
		return []hdfResult{base}
	}

	// The assessment covers all failing resources at once, so repeating it on every
	// location would grow the report quadratically. It travels with the result only
	// when there is a single location to attribute it to.
	res := make([]hdfResult, 0, len(locations))
	for i := range locations {
		loc := hdfLocationLabel(locations[i])
		cur := base
		cur.CodeDesc = codeDesc + " @ " + loc
		cur.Resource = locations[i].Path
		if len(locations) == 1 {
			cur.Message = detail
		} else {
			cur.Message = loc
		}
		res = append(res, cur)
	}
	return res
}

// hdfLocationLabel renders a source context as "path:line".
func hdfLocationLabel(ctx llx.SourceContext) string {
	if startLine, _, _, _, _, ok := ctx.Range.Bounds(); ok && startLine >= 1 {
		return ctx.Path + ":" + strconv.FormatInt(int64(startLine), 10)
	}
	return ctx.Path
}

// hdfStatus maps a check outcome onto an OHDF result status. OHDF has the four
// statuses below and nothing else, so everything that produced neither a verdict
// nor an error - skipped, unscored, unknown - reports as skipped; hdfControlImpact
// then decides whether Heimdall renders it as "Not Applicable" or "Not Reviewed".
func hdfStatus(score *policy.Score) string {
	switch reportdoc.OutcomeOf(score) {
	case reportdoc.OutcomePass:
		return hdfStatusPassed
	case reportdoc.OutcomeFail:
		return hdfStatusFailed
	case reportdoc.OutcomeError:
		return hdfStatusError
	default:
		return hdfStatusSkipped
	}
}

// hdfSkipMessage explains why a check did not run. OHDF has no way to say "skipped
// for no stated reason", so there is always a line: the score's own message when it
// has one, and what kind of score it was when it does not.
//
// The nil case comes first because Score.TypeLabel dereferences without checking.
func hdfSkipMessage(score *policy.Score) string {
	if score == nil {
		return "no score reported"
	}
	if msg := score.MessageLine(); msg != "" {
		return msg
	}
	return "check " + score.TypeLabel()
}

// hdfAssetErrorControl reports an asset that could not be scanned at all. Without
// it the asset would show up as a profile with no controls and read as "nothing to
// check" rather than "nothing was checked".
func hdfAssetErrorControl(ctx *hdfAssetContext, errMsg string) *hdfControl {
	return &hdfControl{
		ID:           hdfAssetErrorID,
		Title:        strPtr("Asset scan error"),
		Desc:         strPtr("cnspec could not complete the scan of this asset. No policy results are available for it."),
		Impact:       1,
		Descriptions: []hdfDescription{},
		Refs:         []hdfRef{},
		Tags: map[string]any{
			"nist":     []string{hdfNistUnmappedTag},
			"severity": "critical",
			"asset":    ctx.asset.Name,
		},
		SourceLocation: hdfSourceLocation{Ref: hdfSourceLocationRef, Line: 1},
		Results: []hdfResult{{
			Status:    hdfStatusError,
			CodeDesc:  "cnspec scan " + hdfAssetLabel(ctx.asset, ctx.assetMrn),
			Message:   errMsg,
			StartTime: ctx.startTime,
		}},
	}
}

// hdfPolicyGroups maps the policies of the bundle onto OHDF control groups, so a
// consumer can still see which policy a control came from even though the profile
// is the asset. Policies with no reporting check on this asset are left out.
func hdfPolicyGroups(bundle *policy.PolicyBundleMap, controlIDs map[string]string) []*hdfGroup {
	res := []*hdfGroup{}
	if bundle == nil || len(controlIDs) == 0 {
		return res
	}

	for _, mrn := range slices.Sorted(maps.Keys(bundle.Policies)) {
		p := bundle.Policies[mrn]
		if p == nil {
			continue
		}

		var ids []string
		seen := map[string]bool{}
		for _, group := range p.Groups {
			if group == nil {
				continue
			}
			for _, check := range group.Checks {
				if check == nil {
					continue
				}
				id, ok := controlIDs[check.Mrn]
				if !ok || seen[id] {
					continue
				}
				seen[id] = true
				ids = append(ids, id)
			}
		}
		if len(ids) == 0 {
			continue
		}
		sort.Strings(ids)

		group := &hdfGroup{ID: mrn, Controls: ids}
		if p.Name != "" {
			group.Title = strPtr(p.Name)
		}
		res = append(res, group)
	}
	return res
}

// hdfVulnControls turns an asset's vulnerability findings into controls: one per
// advisory, plus one catch-all for affected packages that no advisory in the report
// accounts for. They join the asset's own profile rather than forming one of their
// own - see hdfAssetProfile.
func hdfVulnControls(vulnReport *mvd.VulnReport, ctx *hdfAssetContext) []*hdfControl {
	if vulnReport == nil {
		return nil
	}

	affected := map[string]*mvd.Package{}
	byName := map[string]*mvd.Package{}
	for _, pkg := range vulnReport.Packages {
		if pkg == nil || !pkg.Affected {
			continue
		}
		affected[reportdoc.VulnPackageKey(pkg)] = pkg
		byName[pkg.Name] = pkg
	}
	if len(affected) == 0 {
		return nil
	}

	advisories := make([]*mvd.Advisory, 0, len(vulnReport.Advisories))
	for _, advisory := range vulnReport.Advisories {
		if advisory != nil && advisory.ID != "" {
			advisories = append(advisories, advisory)
		}
	}
	sort.Slice(advisories, func(i, j int) bool { return advisories[i].ID < advisories[j].ID })

	var res []*hdfControl
	covered := map[string]bool{}
	for _, advisory := range advisories {
		pkgs := reportdoc.AdvisoryPackages(advisory, affected, byName)
		if len(pkgs) == 0 {
			continue
		}
		for _, pkg := range pkgs {
			covered[reportdoc.VulnPackageKey(pkg)] = true
		}
		res = append(res, hdfAdvisoryControl(ctx, advisory, pkgs))
	}

	var remaining []string
	for key := range affected {
		if !covered[key] {
			remaining = append(remaining, key)
		}
	}
	if len(remaining) > 0 {
		sort.Strings(remaining)
		pkgs := make([]*mvd.Package, 0, len(remaining))
		for _, key := range remaining {
			pkgs = append(pkgs, affected[key])
		}
		res = append(res, hdfAdvisoryControl(ctx, nil, pkgs))
	}
	return res
}

// hdfAdvisoryControl reports one advisory and the packages of this asset it
// affects. With no advisory it reports the packages on their own, which is what
// happens when the report knows a package is vulnerable but carries no advisory
// covering it.
func hdfAdvisoryControl(ctx *hdfAssetContext, advisory *mvd.Advisory, pkgs []*mvd.Package) *hdfControl {
	id := hdfVulnPackageID
	title := "Vulnerable package"
	desc := "An installed package is affected by known vulnerabilities. Update it to a fixed version."
	var score int32
	refs := []hdfRef{}

	if advisory != nil {
		id = advisory.ID
		title = advisory.Title
		if title == "" {
			title = advisory.ID
		}
		desc = strings.TrimSpace(advisory.Description)
		score = advisory.Score
		for _, ref := range advisory.Refs {
			if ref != nil && ref.Url != "" {
				refs = append(refs, hdfRef{Ref: ref.Title, URL: ref.Url})
			}
		}
	}

	tags := map[string]any{
		// SI-2 (flaw remediation) and RA-5 (vulnerability monitoring) are the NIST
		// controls MITRE's converters map vulnerability findings onto.
		"nist":  []string{"SI-2", "RA-5"},
		"asset": ctx.asset.Name,
	}
	if advisory != nil {
		tags["advisory"] = advisory.ID
		if cves := reportdoc.AdvisoryCves(advisory); len(cves) > 0 {
			ids := make([]string, 0, len(cves))
			for _, cve := range cves {
				ids = append(ids, cve.ID)
				if cve.Url != "" {
					refs = append(refs, hdfRef{Ref: cve.ID, URL: cve.Url})
				}
			}
			tags["cves"] = ids
		}
		if advisory.Published != "" {
			tags["published"] = advisory.Published
		}
		if advisory.Modified != "" {
			tags["modified"] = advisory.Modified
		}
	}

	control := &hdfControl{
		ID:             id,
		Title:          strPtr(title),
		Descriptions:   []hdfDescription{},
		Refs:           refs,
		Tags:           tags,
		SourceLocation: hdfSourceLocation{Ref: hdfSourceLocationRef, Line: 1},
		Results:        make([]hdfResult, 0, len(pkgs)),
	}
	if desc != "" {
		control.Desc = strPtr(desc)
	}

	// An advisory without its own score falls back to the worst score of the
	// packages it affects, so the finding never reads as harmless by omission.
	worst := score
	for _, pkg := range pkgs {
		if pkg.Score > worst {
			worst = pkg.Score
		}
	}
	control.Impact = hdfVulnImpact(worst)
	control.Tags["severity"] = hdfSeverityLabel(control.Impact)

	for _, pkg := range pkgs {
		control.Results = append(control.Results, hdfResult{
			Status:    hdfStatusFailed,
			CodeDesc:  pkg.Name + " " + pkg.Version,
			Message:   hdfVulnMessage(advisory, pkg),
			Resource:  pkg.Name,
			StartTime: ctx.startTime,
		})
	}
	return control
}

// hdfVulnImpact maps an advisory or package score (0-100) onto the OHDF scale.
func hdfVulnImpact(score int32) float64 {
	if score >= 100 {
		return 1
	}
	if score <= 0 {
		return hdfMinImpact
	}
	return float64(score) / 100
}

// hdfVulnMessage explains what is wrong with an installed package and what to do
// about it.
func hdfVulnMessage(advisory *mvd.Advisory, pkg *mvd.Package) string {
	msg := pkg.Name + " " + pkg.Version + " has known vulnerabilities"
	if advisory != nil {
		title := advisory.Title
		if title == "" {
			title = advisory.ID
		}
		msg = pkg.Name + " " + pkg.Version + " is affected by " + advisory.ID + " (" + title + ")"
	}
	if pkg.Available != "" {
		return msg + ". Update to " + pkg.Available + "."
	}
	return msg + ". No fixed version is available yet."
}

// hdfPassthroughFor carries the cnspec context that OHDF itself has no field for:
// the asset's identity, platform detail and overall score.
func hdfPassthroughFor(r *policy.ReportCollection, ctx *hdfAssetContext) *hdfPassthrough {
	asset := map[string]any{"mrn": ctx.assetMrn, "name": ctx.asset.Name}
	if platformName := reportdoc.PlatformName(ctx.asset); platformName != "" {
		asset["platform"] = platformName
	}
	if ctx.asset.Platform != nil {
		if ctx.asset.Platform.Version != "" {
			asset["platformVersion"] = ctx.asset.Platform.Version
		}
		if ctx.asset.Platform.Arch != "" {
			asset["arch"] = ctx.asset.Platform.Arch
		}
	}
	if len(ctx.asset.PlatformIds) > 0 {
		asset["platformIds"] = ctx.asset.PlatformIds
	}
	if report := r.Reports[ctx.assetMrn]; report != nil && report.Score != nil {
		asset["score"] = report.Score.Value
		asset["grade"] = report.Score.Rating().Letter()
	}
	if errMsg, ok := r.Errors[ctx.assetMrn]; ok {
		asset["error"] = errMsg
	}

	return &hdfPassthrough{
		AuxiliaryData: []hdfAuxiliaryData{{
			Name: hdfToolName,
			Data: map[string]any{
				"version": cnspec.GetVersion(),
				"asset":   asset,
			},
		}},
	}
}

// hdfNewProfile creates a profile with the OHDF-required fields filled in.
func hdfNewProfile(name string) *hdfProfile {
	return &hdfProfile{
		Name:       name,
		Version:    strPtr(cnspec.GetVersion()),
		Maintainer: strPtr("Mondoo, Inc"),
		Supports:   []map[string]string{},
		Attributes: []any{},
		Depends:    []any{},
		Groups:     []*hdfGroup{},
		Status:     hdfProfileStatus,
		Controls:   []*hdfControl{},
	}
}

// hdfProfileName names the profile after the asset it covers.
func hdfProfileName(assetObj *inventory.Asset, assetMrn string) string {
	if assetObj.Name != "" {
		return assetObj.Name
	}
	return assetMrn
}

// hdfAssetLabel names an asset for human-readable text, qualified by its platform
// when it has one.
func hdfAssetLabel(assetObj *inventory.Asset, assetMrn string) string {
	name := hdfProfileName(assetObj, assetMrn)
	if platformName := reportdoc.PlatformName(assetObj); platformName != "" {
		return name + " (" + platformName + ")"
	}
	return name
}

// hdfAssetSummary describes the scanned platform.
func hdfAssetSummary(assetObj *inventory.Asset) string {
	platformName := reportdoc.PlatformName(assetObj)
	if platformName == "" {
		return "Checks reported by cnspec for this asset"
	}
	if assetObj.Platform != nil && assetObj.Platform.Version != "" {
		return platformName + " " + assetObj.Platform.Version
	}
	return platformName
}

// hdfSupports declares the platform the profile applies to, using the keys InSpec
// profiles use for the same purpose.
func hdfSupports(platform *inventory.Platform) []map[string]string {
	res := []map[string]string{}
	if platform == nil || platform.Name == "" {
		return res
	}

	entry := map[string]string{"platform-name": platform.Name}
	if platform.Version != "" {
		entry["release"] = platform.Version
	}
	return append(res, entry)
}

// hdfProfileChecksum identifies a profile. OHDF requires a sha256 on every profile,
// and consumers treat two profiles carrying the same one as the same profile.
//
// The profile here is the asset, so the asset has to be in the hash. The resolved
// policy's graph checksum alone is not enough: every asset in a fleet scanned
// against the same policies resolves to the same graph, which would hand a hundred
// documents with a hundred different control sets one shared identity. Pairing it
// with the asset keeps the checksum stable across re-scans - which is the property
// a profile checksum is supposed to have - without merging the fleet.
//
// Hashing the control ids is the fallback for a scan that carries no graph
// checksum, e.g. an asset that failed before it resolved anything.
func hdfProfileChecksum(resolved *policy.ResolvedPolicy, key string, controls []*hdfControl) string {
	if resolved != nil && resolved.GraphExecutionChecksum != "" {
		return hdfSha256(key, resolved.GraphExecutionChecksum)
	}

	parts := make([]string, 0, len(controls)+1)
	parts = append(parts, key)
	for _, control := range controls {
		parts = append(parts, control.ID)
	}
	return hdfSha256(parts...)
}

// strPtr is the pointer OHDF's optional string fields are declared as: the schema
// distinguishes a field that is absent from one that is present and empty.
func strPtr(s string) *string {
	return &s
}

func hdfSha256(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:])
}

// hdfRefs maps a check's references onto the OHDF ref shape.
func hdfRefs(refs []*policy.MqueryRef) []hdfRef {
	res := make([]hdfRef, 0, len(refs))
	for _, ref := range refs {
		res = append(res, hdfRef{Ref: ref.Title, URL: ref.Url})
	}
	return res
}

// writeHDFValue emits one JSON document. The output is compact, like the SARIF and
// json-v2 reporters: indenting it costs a second full pass over the buffer and a
// second copy of it, on a document nothing reads by eye. `jq .` renders it.
func writeHDFValue(value any, out iox.OutputHelper) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if _, err := out.Write(data); err != nil {
		return err
	}
	return out.WriteString("\n")
}
