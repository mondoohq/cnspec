// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package exceptions reads exceptions from config files, decides which ones
// govern an asset, and turns them into what the resolver or an upstream acts
// on (ADR-0006).
//
// An exception is read at one of two scopes: the user's client config, or a
// context config, the mondoo.yml a provider found at the root it scanned. A
// context config is written by anyone who can change the scanned repository,
// so it is read through its own restricted parser and only its `exceptions`
// key takes effect.
package exceptions

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/checksums"
	"go.mondoo.com/mql/cli/config"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

const (
	ScopeUser    = "user"
	ScopeContext = "context"

	// expiryWarning is how far ahead an upcoming expiry is reported.
	expiryWarning = 14 * 24 * time.Hour
)

// Actions maps the action vocabulary of a config file to its proto value.
var Actions = map[string]policy.ExceptionAction{
	"disable":        policy.ExceptionAction_EXCEPTION_ACTION_DISABLE,
	"risk-accepted":  policy.ExceptionAction_EXCEPTION_ACTION_RISK_ACCEPTED,
	"false-positive": policy.ExceptionAction_EXCEPTION_ACTION_FALSE_POSITIVE,
	"workaround":     policy.ExceptionAction_EXCEPTION_ACTION_WORKAROUND,
}

// ActionName is the config file spelling of an action.
func ActionName(a policy.ExceptionAction) string {
	for k, v := range Actions {
		if v == a {
			return k
		}
	}
	return "unspecified"
}

// Severity of an Issue.
type Severity int

const (
	// Warning is reported; the entry still applies.
	Warning Severity = iota
	// Error is reported; the entry does not apply.
	Error
)

// Issue is a problem with a config entry.
type Issue struct {
	Severity Severity
	// Source names the file.
	Source string
	// Entry is the entry's position in the file, from 1; 0 for the file.
	Entry   int
	Message string
}

func (i Issue) String() string {
	if i.Entry == 0 {
		return i.Source + ": " + i.Message
	}
	return fmt.Sprintf("%s: exception %d: %s", i.Source, i.Entry, i.Message)
}

// Entry is one entry of a config file, validated.
type Entry struct {
	Source *policy.ExceptionSource
	// Index is the entry's position in the file, from 1.
	Index         int
	Title         string
	Checks        []string
	Paths         []string
	Action        policy.ExceptionAction
	Justification string
	// ValidUntil is zero when the entry does not expire.
	ValidUntil time.Time
}

// Expired reports whether the entry no longer applies at now.
func (e *Entry) Expired(now time.Time) bool {
	return !e.ValidUntil.IsZero() && e.ValidUntil.Before(now)
}

// SourceName names a source for messages and reports.
func SourceName(s *policy.ExceptionSource) string {
	if s == nil {
		return ""
	}
	if s.Repository != "" {
		return s.Repository + "/" + s.Path
	}
	return s.Path
}

// SourceFromOrigin is the source of a context config.
func SourceFromOrigin(o *inventory.ConfigOrigin) *policy.ExceptionSource {
	if o == nil {
		return &policy.ExceptionSource{Scope: ScopeContext}
	}
	return &policy.ExceptionSource{
		Scope:      ScopeContext,
		Provider:   o.Provider,
		Repository: o.Repository,
		Ref:        o.Ref,
		Path:       o.Path,
	}
}

// Parse validates the entries of one config file. Entries with an Error issue
// are left out of the result; the issues say why. now decides what counts as
// expired or close to expiring, which is reported but does not drop an entry.
func Parse(src *policy.ExceptionSource, raw []config.Exception, now time.Time) ([]*Entry, []Issue) {
	name := SourceName(src)
	var res []*Entry
	var issues []Issue
	issue := func(sev Severity, idx int, format string, args ...any) {
		issues = append(issues, Issue{Severity: sev, Source: name, Entry: idx, Message: fmt.Sprintf(format, args...)})
	}

	for i := range raw {
		idx := i + 1
		r := raw[i]
		ok := true

		var checks []string
		for _, c := range r.Checks {
			if c = strings.TrimSpace(c); c != "" {
				checks = append(checks, c)
			}
		}
		if len(checks) == 0 {
			issue(Error, idx, "names no checks")
			ok = false
		}
		for _, c := range checks {
			if strings.ContainsAny(c, "*?[") {
				issue(Error, idx, "check %q: globs are not supported, name each check", c)
				ok = false
			}
		}

		if strings.TrimSpace(r.Justification) == "" {
			issue(Error, idx, "has no justification")
			ok = false
		}

		action, known := Actions[strings.ToLower(strings.TrimSpace(r.Action))]
		switch {
		case known:
		case strings.EqualFold(strings.TrimSpace(r.Action), "out-of-scope"):
			issue(Error, idx, "out-of-scope is a statement about controls, not checks, and is not available here")
			ok = false
		case r.Action == "":
			issue(Error, idx, "has no action, use one of: disable, risk-accepted, false-positive, workaround")
			ok = false
		default:
			issue(Error, idx, "unknown action %q, use one of: disable, risk-accepted, false-positive, workaround", r.Action)
			ok = false
		}

		var validUntil time.Time
		if r.ValidUntil != "" {
			t, err := parseValidUntil(r.ValidUntil)
			if err != nil {
				issue(Error, idx, "valid_until %q is not an RFC3339 date or date-time", r.ValidUntil)
				ok = false
			} else {
				validUntil = t
			}
		}
		if action == policy.ExceptionAction_EXCEPTION_ACTION_DISABLE && !validUntil.IsZero() {
			issue(Warning, idx, "valid_until has no effect on a disabled check")
			validUntil = time.Time{}
		}

		paths, pathIssues := cleanPaths(r.Paths)
		for _, msg := range pathIssues {
			issue(Error, idx, "%s", msg)
			ok = false
		}
		if len(paths) != 0 && src.GetScope() == ScopeUser {
			issue(Error, idx, "paths only apply in a mondoo.yml at a scanned root, not in the client config")
			ok = false
		}

		if !ok {
			continue
		}
		e := &Entry{
			Source:        src,
			Index:         idx,
			Title:         strings.TrimSpace(r.Title),
			Checks:        checks,
			Paths:         paths,
			Action:        action,
			Justification: strings.TrimSpace(r.Justification),
			ValidUntil:    validUntil,
		}
		switch {
		case e.Expired(now):
			issue(Warning, idx, "expired on %s and no longer applies", e.ValidUntil.Format(time.DateOnly))
		case !e.ValidUntil.IsZero() && e.ValidUntil.Sub(now) < expiryWarning:
			issue(Warning, idx, "expires on %s", e.ValidUntil.Format(time.DateOnly))
		}
		res = append(res, e)
	}

	res, conflicts := dropConflicts(res)
	for _, c := range conflicts {
		issue(Error, 0, "%s", c)
	}
	return res, issues
}

func parseValidUntil(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if t, err := time.Parse(time.DateOnly, s); err == nil {
		// a date is valid through the end of that day
		return t.Add(24*time.Hour - time.Second), nil
	}
	return time.Parse(time.RFC3339, s)
}

// cleanPaths normalizes path prefixes and rejects any that could reach outside
// the config's own directory.
func cleanPaths(raw []string) ([]string, []string) {
	var res, issues []string
	for _, p := range raw {
		orig := p
		p = strings.TrimSpace(strings.ReplaceAll(p, "\\", "/"))
		if p == "" {
			issues = append(issues, "has an empty path")
			continue
		}
		if strings.HasPrefix(p, "/") || (len(p) > 1 && p[1] == ':') {
			issues = append(issues, fmt.Sprintf("path %q is absolute; paths are relative to the config's directory", orig))
			continue
		}
		if strings.ContainsAny(p, "*?[") {
			issues = append(issues, fmt.Sprintf("path %q: globs are not supported, use a path prefix", orig))
			continue
		}
		escapes := false
		for _, seg := range strings.Split(p, "/") {
			if seg == ".." {
				escapes = true
			}
		}
		if escapes {
			issues = append(issues, fmt.Sprintf("path %q leaves the config's directory", orig))
			continue
		}
		res = append(res, path.Clean(p))
	}
	sort.Strings(res)
	return res, issues
}

// dropConflicts removes entries that name the same check for the same path
// scope with different actions. At equal specificity there is no right answer,
// so neither applies and the file has to be fixed.
func dropConflicts(entries []*Entry) ([]*Entry, []string) {
	type slot struct {
		check string
		path  string
	}
	seen := map[slot]*Entry{}
	bad := map[*Entry]struct{}{}
	var msgs []string
	for _, e := range entries {
		scopes := e.Paths
		if len(scopes) == 0 {
			scopes = []string{""}
		}
		for _, c := range e.Checks {
			for _, p := range scopes {
				k := slot{c, p}
				prev, ok := seen[k]
				if !ok {
					seen[k] = e
					continue
				}
				if prev.Action == e.Action {
					continue
				}
				where := "every path"
				if p != "" {
					where = "path " + p
				}
				msgs = append(msgs, fmt.Sprintf("exceptions %d and %d give check %s different actions for %s; neither applies",
					prev.Index, e.Index, c, where))
				bad[prev] = struct{}{}
				bad[e] = struct{}{}
			}
		}
	}
	if len(bad) == 0 {
		return entries, nil
	}
	var res []*Entry
	for _, e := range entries {
		if _, ok := bad[e]; !ok {
			res = append(res, e)
		}
	}
	return res, msgs
}

// Match is an entry that governs an asset, for one check.
type Match struct {
	Entry *Entry
	Check string
	// MatchedPath is the path prefix that matched, "" for an unscoped entry.
	MatchedPath string
}

// MatchAsset returns, per check, the entry that governs an asset at assetPath,
// relative to the config's directory. An asset with no path ("") is governed
// only by unscoped entries. Where several entries name a check, the one with
// the longest matching prefix wins; an unscoped entry is the least specific.
func MatchAsset(entries []*Entry, assetPath string) []Match {
	type best struct {
		m    Match
		spec int
	}
	byCheck := map[string]best{}
	var order []string

	for _, e := range entries {
		spec, matched, ok := matchPaths(e.Paths, assetPath)
		if !ok {
			continue
		}
		for _, c := range e.Checks {
			cur, exists := byCheck[c]
			if !exists {
				order = append(order, c)
			}
			if !exists || spec > cur.spec {
				byCheck[c] = best{m: Match{Entry: e, Check: c, MatchedPath: matched}, spec: spec}
			}
		}
	}

	res := make([]Match, 0, len(order))
	for _, c := range order {
		res = append(res, byCheck[c].m)
	}
	return res
}

// matchPaths matches an asset path against an entry's path prefixes on
// directory boundaries. It returns the specificity of the match: -1 for an
// unscoped entry, otherwise the length of the longest matching prefix.
func matchPaths(prefixes []string, assetPath string) (int, string, bool) {
	if len(prefixes) == 0 {
		return -1, "", true
	}
	if assetPath == "" {
		return 0, "", false
	}
	assetPath = path.Clean(assetPath)
	spec, matched, ok := 0, "", false
	for _, p := range prefixes {
		if p == "." || assetPath == p || strings.HasPrefix(assetPath, p+"/") {
			if l := len(p); !ok || l > spec {
				spec, matched, ok = l, p, true
			}
		}
	}
	return spec, matched, ok
}

// EntryKey identifies a check exception within its source, from its path scope,
// check and action, so it stays stable when entries are reordered.
func EntryKey(paths []string, check string, action policy.ExceptionAction) string {
	sum := checksums.New.Add(check).Add(ActionName(action))
	for _, p := range paths {
		sum = sum.Add(p)
	}
	return sum.String()
}

// CheckResolver turns a check as a file names it, a UID or an MRN, into the
// check's MRN. It returns "" for a check that is not known.
type CheckResolver func(check string) string

// ToProto turns a match into the entry that is submitted and reported. The
// check is resolved to its MRN; an unknown check keeps an empty check_mrn.
func ToProto(m Match, resolve CheckResolver) *policy.ExceptionEntry {
	e := m.Entry
	res := &policy.ExceptionEntry{
		Key:           EntryKey(e.Paths, m.Check, e.Action),
		Action:        e.Action,
		Title:         e.Title,
		Justification: e.Justification,
		Paths:         e.Paths,
		MatchedPath:   m.MatchedPath,
		Source:        e.Source,
	}
	if strings.HasPrefix(m.Check, "//") {
		res.CheckMrn = m.Check
	} else {
		res.CheckUid = m.Check
		if resolve != nil {
			res.CheckMrn = resolve(m.Check)
		}
	}
	if !e.ValidUntil.IsZero() {
		res.ValidUntil = &policy.HumanTime{Seconds: e.ValidUntil.Unix()}
	}
	return res
}

// AllKeys lists every key a source's entries produce, for declaring the
// source's complete set upstream.
func AllKeys(entries []*Entry) []string {
	var res []string
	for _, e := range entries {
		for _, c := range e.Checks {
			res = append(res, EntryKey(e.Paths, c, e.Action))
		}
	}
	sort.Strings(res)
	return res
}

// SetChecksum is a checksum over a source's normalized entries. It changes
// when any entry changes, including its justification or expiry.
func SetChecksum(entries []*Entry) string {
	keys := make([]string, 0, len(entries))
	for _, e := range entries {
		k := checksums.New.Add(e.Title).Add(e.Justification).Add(ActionName(e.Action))
		if !e.ValidUntil.IsZero() {
			k = k.Add(e.ValidUntil.UTC().Format(time.RFC3339))
		}
		for _, c := range e.Checks {
			k = k.Add("c:" + c)
		}
		for _, p := range e.Paths {
			k = k.Add("p:" + p)
		}
		keys = append(keys, k.String())
	}
	sort.Strings(keys)
	sum := checksums.New
	for _, k := range keys {
		sum = sum.Add(k)
	}
	return sum.String()
}

// GroupType is the policy group type an action becomes.
func GroupType(a policy.ExceptionAction) policy.GroupType {
	if a == policy.ExceptionAction_EXCEPTION_ACTION_DISABLE {
		return policy.GroupType_DISABLE
	}
	return policy.GroupType_IGNORED
}

// PolicyGroups turns entries that apply into the exception groups the resolver
// reads: one group per entry, so each keeps its own justification and expiry.
// Entries without a resolved check are left out.
func PolicyGroups(entries []*policy.ExceptionEntry) []*policy.PolicyGroup {
	var res []*policy.PolicyGroup
	for _, e := range entries {
		if e.CheckMrn == "" {
			continue
		}
		g := &policy.PolicyGroup{
			Uid:    "exception-" + e.Key,
			Type:   GroupType(e.Action),
			Title:  e.Title,
			Checks: []*policy.Mquery{{Mrn: e.CheckMrn}},
			Docs:   &policy.PolicyGroupDocs{Justification: e.Justification},
		}
		if e.ValidUntil != nil {
			g.Valid = &policy.Validity{Until: e.ValidUntil}
		}
		res = append(res, g)
	}
	return res
}
