// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package textrank

import (
	"strings"
	"unicode"
)

// stopwords are common English and policy-boilerplate terms that carry no
// discriminating signal for ranking.
var stopwords = map[string]bool{
	"a": true, "an": true, "and": true, "are": true, "as": true, "at": true,
	"be": true, "but": true, "by": true, "for": true, "from": true, "has": true,
	"have": true, "in": true, "is": true, "it": true, "its": true, "must": true,
	"not": true, "of": true, "on": true, "or": true, "should": true, "that": true,
	"the": true, "their": true, "them": true, "then": true, "there": true,
	"these": true, "this": true, "to": true, "using": true, "via": true, "was": true,
	"were": true, "which": true, "will": true, "with": true, "all": true,
	"any": true, "each": true, "when": true, "where": true, "ensure": true,
	"check": true, "enabled": true, "disabled": true, "configured": true,
}

// tokenize splits text into lowercase, stemmed word tokens, dropping stopwords
// and single characters. It splits on any non-alphanumeric rune so identifiers
// like "aws.s3.buckets" and "server-side" break into their parts.
func tokenize(s string) []string {
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		// keep 2-char tokens (s3, ip, ci, vm, db) which are meaningful resource
		// shorthands; drop single characters and stopwords.
		if len(f) < 2 || stopwords[f] {
			continue
		}
		out = append(out, stem(f))
	}
	return out
}

// stem applies a light, conservative suffix stripper so that "buckets"/"bucket",
// "policies"/"policy", and "encrypted"/"encrypt" match. It is deliberately
// simple (not a full Porter stemmer) — good enough to improve recall without a
// dependency, and it never touches short tokens where stripping would be noise.
// It is applied identically at index and query time.
func stem(w string) string {
	switch {
	case len(w) > 4 && strings.HasSuffix(w, "ies"):
		return w[:len(w)-3] + "y" // policies -> policy
	case len(w) > 4 && (strings.HasSuffix(w, "sses") || strings.HasSuffix(w, "ches") ||
		strings.HasSuffix(w, "shes") || strings.HasSuffix(w, "xes") || strings.HasSuffix(w, "zes")):
		return w[:len(w)-2] // classes -> class
	case len(w) > 5 && strings.HasSuffix(w, "ing"):
		return w[:len(w)-3] // encrypting -> encrypt
	case len(w) > 4 && strings.HasSuffix(w, "ed"):
		return w[:len(w)-2] // encrypted -> encrypt
	case len(w) > 3 && strings.HasSuffix(w, "s") && !strings.HasSuffix(w, "ss"):
		return w[:len(w)-1] // buckets -> bucket
	}
	return w
}
