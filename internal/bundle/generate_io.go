// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package bundle

import "strings"

// AllQueries returns pointers to every query definition in a bundle, in a
// stable order: top-level shared queries first, then checks and data queries in
// policy groups, then query-pack queries. Callers can read intent fields and
// write Mql back through these pointers; changes are reflected when the bundle
// is re-formatted via FormatBundle.
//
// This is the seam used by `cnspec policy generate` — it keeps knowledge of the
// comment-preserving internal bundle structs in this package so the command and
// the generator stay decoupled from the YAML layer.
func AllQueries(b *Bundle) []*Mquery {
	if b == nil {
		return nil
	}
	var out []*Mquery
	out = append(out, b.Queries...)
	for _, p := range b.Policies {
		if p == nil {
			continue
		}
		for _, g := range p.Groups {
			if g == nil {
				continue
			}
			out = append(out, g.Checks...)
			out = append(out, g.Queries...)
		}
	}
	for _, pack := range b.Packs {
		if pack == nil {
			continue
		}
		out = append(out, pack.Queries...)
		for _, g := range pack.Groups {
			if g == nil {
				continue
			}
			out = append(out, g.Queries...)
		}
	}
	return out
}

// QueryDesc returns a query's description, preferring docs.desc and falling back
// to the legacy top-level desc.
func QueryDesc(q *Mquery) string {
	if q == nil {
		return ""
	}
	if q.Docs != nil && strings.TrimSpace(q.Docs.Desc) != "" {
		return q.Docs.Desc
	}
	return q.Desc
}

// QueryFilterStrings returns the filter MQL expressions attached to a query.
func QueryFilterStrings(q *Mquery) []string {
	if q == nil || q.Filters == nil {
		return nil
	}
	var out []string
	for _, f := range q.Filters.Items {
		if f != nil && strings.TrimSpace(f.Mql) != "" {
			out = append(out, f.Mql)
		}
	}
	return out
}

// QueryHasVariants reports whether a query delegates its MQL to per-platform
// variants (so the parent must not receive generated MQL).
func QueryHasVariants(q *Mquery) bool {
	return q != nil && len(q.Variants) > 0
}

// VariantParents maps each variant child's uid to the parent query that declares
// it. A variant leaf usually carries only filters + mql and inherits its intent
// (title/description) from the parent, so the generator uses this to give leaves
// something to generate from.
func VariantParents(b *Bundle) map[string]*Mquery {
	out := map[string]*Mquery{}
	if b == nil {
		return out
	}
	for _, q := range AllQueries(b) {
		for _, v := range q.Variants {
			if v != nil && v.Uid != "" {
				out[v.Uid] = q
			}
		}
	}
	return out
}

// QueryUIDs returns the set of uids already defined by queries in a bundle —
// the top-level `queries:` block, policy groups, and query packs.
//
// `cnspec policy generate` looks this up before it adds a check: two queries
// sharing a uid in one file is a lint error (`query-uid-unique`), and a bundle
// that fails lint is a worse outcome than re-asking for the uid.
func QueryUIDs(b *Bundle) map[string]bool {
	out := map[string]bool{}
	for _, q := range AllQueries(b) {
		if q != nil && q.Uid != "" {
			out[q.Uid] = true
		}
	}
	return out
}

// SanitizeText applies the same normalization FormatBundle performs on a string
// it writes into a bundle: tabs expanded, CR folded into LF, trailing whitespace
// trimmed, and every non-graphic character (NUL, ANSI escape sequences, …)
// removed.
//
// Model-authored text reaches the user twice — once on the review screen, once
// in the file — and only the terminal can be lied to: an ESC[2K / ESC[1G pair
// inside a generated query repaints the line the reviewer is approving, so what
// was reviewed and what was committed are not the same text. Rendering through
// this function makes the review screen show exactly the bytes FormatBundle
// will write, which is the assumption the human-in-the-loop gate rests on.
func SanitizeText(s string) string {
	return sanitizeStringForYaml(s)
}
