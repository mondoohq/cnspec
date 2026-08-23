// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package reportmodel turns a scan result (*policy.ReportCollection) into a
// navigable asset -> policy -> check tree.
//
// It is a data model, not a renderer: nothing here writes to a terminal, picks a
// color or knows about bubbletea. A viewer walks Report.Assets, then
// Asset.Policies, then Policy.Checks, and asks a Check for its Detail when the
// user opens it.
//
// Two things this package exists to get right, because both are easy to get
// wrong when reading a report collection directly:
//
//   - The checks that ran on an asset are not the entries of Report.Scores. That
//     map mixes policy scores, control scores, check scores (keyed by MRN) and
//     execution-query scores (keyed by code id). The checks come from the
//     asset's ResolvedPolicy: the CHECK / CHECK_AND_DATA_QUERY reporting jobs,
//     each confirmed against CollectorJob.ReportingQueries so we only surface
//     checks that actually ran on this asset's platform.
//
//   - "Failed" is not a score type. It is a result score whose value is below
//     100, so failing, errored, skipped and unscored checks are four distinct
//     outcomes (see Status). An asset that failed to scan is likewise not an
//     asset without findings: it has no report and no resolved policy at all,
//     and surfaces here as StatusError with its ScanError set.
package reportmodel

import (
	"regexp"
	"sort"
	"strings"

	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/reports/reportdoc"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

// UngroupedPolicyName is the name of the synthetic policy node that collects the
// checks of an asset whose reporting job does not lead up to any policy in the
// bundle. It is always sorted last and has an empty Mrn.
const UngroupedPolicyName = "Other checks"

// Status is the outcome of a check, a policy or an asset. The six values mirror
// reportdoc.Outcome, which is the single place in cnspec that collapses the
// seven score types into an outcome.
//
// Note that StatusFail and StatusPass share a score type (ScoreType_Result) and
// are told apart by the score value, while StatusError, StatusSkipped and
// StatusUnscored are score types of their own. A viewer must not collapse them:
// a check that could not run is not a check that passed.
type Status string

const (
	StatusPass     Status = "PASS"
	StatusFail     Status = "FAIL"
	StatusError    Status = "ERROR"
	StatusSkipped  Status = "SKIPPED"
	StatusUnscored Status = "UNSCORED"
	StatusUnknown  Status = "UNKNOWN"
)

// StatusOf maps a score to its outcome. It is nil-safe: a missing score is
// StatusUnknown, never StatusPass.
func StatusOf(score *policy.Score) Status {
	return Status(reportdoc.OutcomeOf(score).Label())
}

// Icon is the emoji for a status. It matches the SARIF reporter's icons.
func (s Status) Icon() string {
	switch s {
	case StatusPass:
		return "✅"
	case StatusFail:
		return "❌"
	case StatusError:
		return "⚠️"
	case StatusSkipped:
		return "⏭️"
	default:
		return "ℹ️"
	}
}

// IsFinding reports whether the status is something the user has to act on: a
// check that failed, or one that errored and therefore proved nothing.
func (s Status) IsFinding() bool {
	return s == StatusFail || s == StatusError
}

// Counts is the tally of outcomes over a set of checks or assets. Total is the
// number counted, not the sum of the other fields, so an unrecognized status is
// still counted once.
type Counts struct {
	Total    int
	Passed   int
	Failed   int
	Errored  int
	Skipped  int
	Unscored int
	Unknown  int
}

// Add counts one outcome.
func (c *Counts) Add(s Status) {
	c.Total++
	switch s {
	case StatusPass:
		c.Passed++
	case StatusFail:
		c.Failed++
	case StatusError:
		c.Errored++
	case StatusSkipped:
		c.Skipped++
	case StatusUnscored:
		c.Unscored++
	default:
		c.Unknown++
	}
}

// Findings is the number of outcomes that need action: failed plus errored.
func (c Counts) Findings() int {
	return c.Failed + c.Errored
}

// PolicyRef names a policy a check belongs to.
type PolicyRef struct {
	// Mrn is the policy MRN, empty for the synthetic UngroupedPolicyName node.
	Mrn string
	// Name is the human-readable policy title, falling back to the MRN.
	Name string
}

// Report is the root of the model: every asset the scan touched, whether or not
// it produced a report.
type Report struct {
	// Assets are all assets of the scan, sorted by name and then MRN.
	Assets []*Asset
	// AssetCounts tallies the outcome of each asset.
	AssetCounts Counts
	// CheckCounts tallies the outcome of every check on every asset. A check
	// that ran on two assets is counted twice.
	CheckCounts Counts

	collection *policy.ReportCollection
	byMrn      map[string]*Asset
}

// Asset is one scanned entity, with the policies and checks that ran on it.
type Asset struct {
	// Mrn identifies the asset and is the key into the report collection.
	Mrn string
	// Name is the human-readable asset name, falling back to the MRN.
	Name string
	// Platform is the platform the asset was detected as. It is nil for an
	// asset that failed to scan before detection completed.
	Platform *inventory.Platform
	// PlatformName is the platform title, falling back to its name.
	PlatformName string
	// Score is the asset's overall score. For an asset that never produced a
	// report it is synthesized: a ScoreType_Error score carrying ScanError, or
	// a ScoreType_Unknown score when there is no error either.
	Score *policy.Score
	// Status is the asset's outcome.
	Status Status
	// ScanError is the scan failure of this asset, empty when it scanned. An
	// asset with a ScanError has no policies and no checks - that is a scan
	// that did not happen, not a scan without findings.
	ScanError string
	// Policies are the policies that ran on this asset, sorted by name; the
	// synthetic UngroupedPolicyName node, if any, is last.
	Policies []*Policy
	// Checks are all checks of this asset, each once, sorted by title and then
	// MRN. A check that belongs to two policies appears once here and in both
	// Policy.Checks.
	Checks []*Check
	// Counts tallies Checks.
	Counts Counts

	report   *policy.Report
	resolved *policy.ResolvedPolicy
	// platformKeys filters remediation down to this asset's platform.
	platformKeys map[string]bool
}

// Scanned reports whether the asset produced a report at all.
func (a *Asset) Scanned() bool {
	return a != nil && a.report != nil
}

// Policy is one policy as it applied to a single asset.
type Policy struct {
	// Mrn is the policy MRN, empty for the synthetic UngroupedPolicyName node.
	Mrn string
	// Name is the policy title, falling back to the MRN.
	Name string
	// Score is the policy's score on this asset; nil when it has none.
	Score *policy.Score
	// Status is the policy's outcome on this asset.
	Status Status
	// Checks are the checks of this policy on this asset, sorted by title and
	// then MRN.
	Checks []*Check
	// Counts tallies Checks.
	Counts Counts
}

// Check is one check as it ran on a single asset.
type Check struct {
	// Mrn is the check MRN. It is empty only for a check that has no MRN at
	// all, in which case CodeId identifies it.
	Mrn string
	// CodeId is the checksum of the compiled MQL. It is what the collector job
	// keys reporting queries by.
	CodeId string
	// Title is the check title, falling back to the MRN or the code id.
	Title string
	// Score is the check's score on this asset, nil when the check reported
	// nothing.
	Score *policy.Score
	// Status is the check's outcome: PASS, FAIL, ERROR, SKIPPED, UNSCORED or
	// UNKNOWN.
	Status Status
	// Impact is the configured impact of the check (0-100, 100 being the most
	// impactful) and HasImpact says whether it is configured at all.
	Impact    int32
	HasImpact bool
	// Severity is the label of Impact (CRITICAL/HIGH/MEDIUM/LOW/NONE). It is
	// what the check is worth, not how it did - use Risk for the latter.
	Severity string
	// Risk is the realized risk of the score (100 - value): 0 for a check that
	// passed, 100 for one that failed outright. A score type that carries no
	// value - error, skip, unscored - has value 0 and therefore risk 100, so
	// read Risk together with Status rather than on its own.
	Risk int32
	// Policies are the policies of this asset that include the check, sorted by
	// name.
	Policies []PolicyRef
	// Query is the check itself. It is never nil.
	Query *policy.Mquery

	asset *Asset
}

// New builds the model from a report collection. The collection is not
// modified and may be nil, in which case the result is an empty report.
func New(collection *policy.ReportCollection) *Report {
	res := &Report{
		collection: collection,
		byMrn:      map[string]*Asset{},
	}
	if collection == nil {
		return res
	}

	b := newBuilder(collection)
	for _, mrn := range b.assetMrns() {
		asset := b.buildAsset(mrn)
		res.Assets = append(res.Assets, asset)
		res.byMrn[mrn] = asset
		res.AssetCounts.Add(asset.Status)
		for _, check := range asset.Checks {
			res.CheckCounts.Add(check.Status)
		}
	}

	sort.SliceStable(res.Assets, func(i, j int) bool {
		if res.Assets[i].Name != res.Assets[j].Name {
			return res.Assets[i].Name < res.Assets[j].Name
		}
		return res.Assets[i].Mrn < res.Assets[j].Mrn
	})

	return res
}

// Collection returns the report collection the model was built from, so a
// viewer can reach data the model does not cover (vulnerability reports, raw
// results). It may be nil.
func (r *Report) Collection() *policy.ReportCollection {
	return r.collection
}

// Asset looks an asset up by MRN, returning nil when there is none.
func (r *Report) Asset(mrn string) *Asset {
	return r.byMrn[mrn]
}

// builder carries the bundle-derived lookups shared by every asset.
type builder struct {
	collection *policy.ReportCollection
	bundle     *policy.PolicyBundleMap
	// queriesByMrn is the bundle's queries keyed by MRN, which is what a
	// reporting job's QrId normally is.
	queriesByMrn map[string]*policy.Mquery
	// queriesByCodeID is the same set keyed by code id. It resolves the jobs of
	// an unsigned bundle, whose QrId is a code id rather than an MRN.
	queriesByCodeID map[string]*policy.Mquery
	policyNames     map[string]string
}

func newBuilder(collection *policy.ReportCollection) *builder {
	b := &builder{
		collection:      collection,
		queriesByMrn:    map[string]*policy.Mquery{},
		queriesByCodeID: map[string]*policy.Mquery{},
		policyNames:     map[string]string{},
	}

	// Bundle is nil for a scan that failed before resolving policies - the
	// all-errored k8s report is exactly that - so this cannot be unconditional.
	if collection.Bundle == nil {
		return b
	}

	b.bundle = collection.Bundle.ToMap()
	b.queriesByMrn = b.bundle.Queries
	// DeterministicQueryMap rather than PolicyBundleMap.QueryMap: the latter
	// picks an arbitrary query when variants share a code id.
	b.queriesByCodeID = reportdoc.QueryMap(b.bundle)
	for mrn, p := range b.bundle.Policies {
		if p == nil {
			continue
		}
		name := p.Name
		if name == "" {
			name = mrn
		}
		b.policyNames[mrn] = name
	}

	return b
}

// assetMrns is every asset the collection knows about: the inventory, plus any
// MRN that only shows up as a report or as an error, so an asset can never go
// missing from the model.
func (b *builder) assetMrns() []string {
	seen := map[string]struct{}{}
	for mrn := range b.collection.Assets {
		seen[mrn] = struct{}{}
	}
	for mrn := range b.collection.Reports {
		seen[mrn] = struct{}{}
	}
	for mrn := range b.collection.Errors {
		seen[mrn] = struct{}{}
	}
	return sortedKeys(seen)
}

func (b *builder) buildAsset(mrn string) *Asset {
	asset := &Asset{
		Mrn:       mrn,
		Name:      mrn,
		ScanError: plainError(b.collection.Errors[mrn]),
		report:    b.collection.Reports[mrn],
		resolved:  b.collection.ResolvedPolicies[mrn],
	}

	if inv := b.collection.Assets[mrn]; inv != nil {
		if inv.Name != "" {
			asset.Name = inv.Name
		}
		asset.Platform = inv.Platform
		if p := inv.Platform; p != nil {
			asset.PlatformName = p.Title
			if asset.PlatformName == "" {
				asset.PlatformName = p.Name
			}
		}
	}
	asset.platformKeys = reportdoc.PlatformRemediationKeys(asset.Platform)

	asset.Score = b.assetScore(mrn, asset.report)
	asset.Status = StatusOf(asset.Score)

	b.buildChecks(asset)
	return asset
}

// assetScore is the asset's overall score, synthesized when the asset produced
// no report. This mirrors reporter.GenerateStats: an asset that failed to scan
// gets a ScoreType_Error score carrying the scan error, so a single code path
// covers reports, errors and assets with neither. (GenerateStats itself is not
// called here because it dereferences ReportCollection.Bundle, which is nil for
// precisely the all-errored collections this has to handle.)
func (b *builder) assetScore(mrn string, report *policy.Report) *policy.Score {
	if report != nil {
		if score, ok := report.Scores[mrn]; ok && score != nil {
			return score
		}
		if report.Score != nil {
			return report.Score
		}
	}
	if msg := plainError(b.collection.Errors[mrn]); msg != "" {
		return &policy.Score{
			QrId:    mrn,
			Type:    policy.ScoreType_Error,
			Message: msg,
		}
	}
	return &policy.Score{
		QrId: mrn,
		Type: policy.ScoreType_Unknown,
	}
}

// buildChecks fills in the asset's checks and policies from its resolved policy.
//
// The checks of an asset are the CHECK and CHECK_AND_DATA_QUERY reporting jobs
// of its collector job - not the entries of Report.Scores, which also hold
// policy, control and execution-query scores. Each job's QrId is resolved
// against the bundle and then confirmed against ReportingQueries, because the
// bundle holds queries that do not match this asset's platform.
func (b *builder) buildChecks(asset *Asset) {
	if asset.resolved == nil || asset.resolved.CollectorJob == nil {
		return
	}
	jobs := asset.resolved.CollectorJob.ReportingJobs
	reportingQueries := asset.resolved.CollectorJob.ReportingQueries

	byKey := map[string]*Check{}
	policies := map[string]*Policy{}

	for _, uuid := range sortedKeys(jobs) {
		job := jobs[uuid]
		if job == nil {
			continue
		}
		if job.Type != policy.ReportingJob_CHECK && job.Type != policy.ReportingJob_CHECK_AND_DATA_QUERY {
			continue
		}

		query := b.query(job.QrId)
		if query == nil || query.CodeId == "" {
			continue
		}
		// Confirm the check actually ran on this asset.
		if _, ok := reportingQueries[query.CodeId]; !ok {
			continue
		}

		key := query.Mrn
		if key == "" {
			key = query.CodeId
		}

		check, ok := byKey[key]
		if !ok {
			check = b.buildCheck(asset, job, query)
			byKey[key] = check
			asset.Checks = append(asset.Checks, check)
			asset.Counts.Add(check.Status)
		}

		for _, policyMrn := range b.policyAncestors(jobs, job) {
			b.attach(policies, policyMrn, check)
		}
	}

	// A check whose reporting job leads up to no policy in the bundle still has
	// to be reachable in the tree.
	for _, check := range asset.Checks {
		if len(check.Policies) == 0 {
			b.attach(policies, "", check)
		}
	}

	sortChecks(asset.Checks)
	for _, check := range asset.Checks {
		refs := check.Policies
		sort.SliceStable(refs, func(i, j int) bool {
			if refs[i].Name != refs[j].Name {
				return refs[i].Name < refs[j].Name
			}
			return refs[i].Mrn < refs[j].Mrn
		})
	}

	for _, mrn := range sortedKeys(policies) {
		p := policies[mrn]
		sortChecks(p.Checks)
		for _, check := range p.Checks {
			p.Counts.Add(check.Status)
		}
		if asset.report != nil && p.Mrn != "" {
			p.Score = asset.report.Scores[p.Mrn]
		}
		p.Status = StatusOf(p.Score)
		asset.Policies = append(asset.Policies, p)
	}

	sort.SliceStable(asset.Policies, func(i, j int) bool {
		a, c := asset.Policies[i], asset.Policies[j]
		// the synthetic node sorts last
		if (a.Mrn == "") != (c.Mrn == "") {
			return c.Mrn == ""
		}
		if a.Name != c.Name {
			return a.Name < c.Name
		}
		return a.Mrn < c.Mrn
	})
}

func (b *builder) attach(policies map[string]*Policy, policyMrn string, check *Check) {
	p, ok := policies[policyMrn]
	if !ok {
		name := b.policyNames[policyMrn]
		if name == "" {
			name = policyMrn
		}
		if policyMrn == "" {
			name = UngroupedPolicyName
		}
		p = &Policy{Mrn: policyMrn, Name: name}
		policies[policyMrn] = p
	}
	p.Checks = append(p.Checks, check)
	check.Policies = append(check.Policies, PolicyRef{Mrn: p.Mrn, Name: p.Name})
}

func (b *builder) buildCheck(asset *Asset, job *policy.ReportingJob, query *policy.Mquery) *Check {
	score := b.checkScore(asset.report, job, query)

	title := query.Title
	if title == "" {
		title = query.Mrn
	}
	if title == "" {
		title = query.CodeId
	}

	check := &Check{
		Mrn:    query.Mrn,
		CodeId: query.CodeId,
		Title:  title,
		Score:  score,
		Status: StatusOf(score),
		Risk:   reportdoc.ScoreRisk(score),
		Query:  query,
		asset:  asset,
	}
	check.Impact, check.HasImpact = reportdoc.QueryImpact(query)
	check.Severity = reportdoc.RiskSeverityLabel(check.Impact)
	return check
}

// checkScore looks a check's score up. Both keys are in use depending on how the
// score was collected, so try the MRN first and fall back to the code id.
//
// The score is returned as it is stored. The compact reporter raises the value
// of a low-impact failing check in place (a documented FIXME v12 workaround);
// this model does not mutate scores, so a viewer sees what the scan recorded.
func (b *builder) checkScore(report *policy.Report, job *policy.ReportingJob, query *policy.Mquery) *policy.Score {
	if report == nil {
		return nil
	}
	for _, key := range []string{query.Mrn, job.QrId, query.CodeId} {
		if key == "" {
			continue
		}
		if score, ok := report.Scores[key]; ok && score != nil {
			return score
		}
	}
	return nil
}

// query resolves a reporting job's QrId to a check. A signed bundle keys its
// queries by MRN, which is what QrId holds; an unsigned one has no MRNs and uses
// the code id, hence the second lookup.
func (b *builder) query(qrID string) *policy.Mquery {
	if qrID == "" {
		return nil
	}
	if query, ok := b.queriesByMrn[qrID]; ok && query != nil {
		return query
	}
	return b.queriesByCodeID[qrID]
}

// policyAncestors walks up the reporting graph from a check and returns the MRNs
// of the nearest policies that own it. It stops at the first policy job on each
// branch, so a check of a policy that is itself bundled into a parent policy (or
// into the space) is attributed to the policy that declares it, not to every
// ancestor of it.
func (b *builder) policyAncestors(jobs map[string]*policy.ReportingJob, job *policy.ReportingJob) []string {
	var res []string
	seen := map[string]struct{}{}
	queue := append([]string{}, job.Notify...)

	for i := 0; i < len(queue); i++ {
		uuid := queue[i]
		if _, ok := seen[uuid]; ok {
			continue
		}
		seen[uuid] = struct{}{}

		parent := jobs[uuid]
		if parent == nil {
			continue
		}
		if parent.Type == policy.ReportingJob_POLICY {
			if _, known := b.policyNames[parent.QrId]; known {
				res = append(res, parent.QrId)
				continue
			}
		}
		queue = append(queue, parent.Notify...)
	}

	sort.Strings(res)
	return res
}

func sortChecks(checks []*Check) {
	sort.SliceStable(checks, func(i, j int) bool {
		if checks[i].Title != checks[j].Title {
			return checks[i].Title < checks[j].Title
		}
		return checks[i].Mrn < checks[j].Mrn
	})
}

func sortedKeys[T any](m map[string]T) []string {
	res := make([]string, 0, len(m))
	for k := range m {
		res = append(res, k)
	}
	sort.Strings(res)
	return res
}

// normalizeNewlines strips carriage returns so a string carried into a
// width-sensitive renderer cannot corrupt its line maths. Reporter output uses
// NewLineCharacter, which is "\r\n" on Windows.
func normalizeNewlines(s string) string {
	if !strings.Contains(s, "\r") {
		return s
	}
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}

// rpcEnvelope matches the wrapper gRPC puts around a provider's own words:
// "rpc error: code = InvalidArgument desc = asset doesn't support any
// policies". The code is a transport detail; the sentence after desc= is the
// part a reader can act on, and in the fixtures every single scan error
// carries this prefix.
var rpcEnvelope = regexp.MustCompile(`^rpc error: code = \S+ desc = `)

// plainError strips that envelope, leaving the message the provider wrote.
func plainError(msg string) string {
	return strings.TrimSpace(rpcEnvelope.ReplaceAllString(strings.TrimSpace(msg), ""))
}
