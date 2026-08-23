// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generate

import (
	"os"
	"strings"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnspec/internal/textrank"
	"go.mondoo.com/cnspec/policy"
)

// Field weights and the provider match bonus for grounding search. Title matches
// dominate because a check's title is the most concentrated statement of intent;
// a same-provider example is what the agent most needs, so it gets a large bonus
// without excluding cross-provider matches.
const (
	weightTitle   = 3
	weightDesc    = 1
	weightMql     = 1
	providerBonus = 5.0
)

// Example is one existing, validated check used to ground generation. Real MQL
// for the same provider is the single biggest accuracy lever we have, so the
// generator feeds the closest examples to the agent as few-shot context.
type Example struct {
	UID      string
	Title    string
	Desc     string
	Mql      string
	Provider string
	// Score is the similarity to the query intent, set by Search.
	Score float64
}

// Corpus is a searchable set of examples drawn from existing policy bundles
// (content/*.mql.yaml by default), ranked with the shared BM25 text ranker.
type Corpus struct {
	byUID map[string]Example
	index *textrank.Index
}

// LoadCorpus reads every query with MQL from the given bundle paths and builds a
// searchable corpus.
func LoadCorpus(paths ...string) (*Corpus, error) {
	files, err := policy.WalkPolicyBundleFiles(paths...)
	if err != nil {
		return nil, errors.Wrap(err, "could not enumerate corpus files")
	}

	var examples []Example
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue // a single unreadable file should not sink the corpus
		}
		b, err := policy.BundleFromYAML(data)
		if err != nil {
			continue
		}
		for _, q := range allBundleQueries(b) {
			if strings.TrimSpace(q.Mql) == "" {
				continue
			}
			examples = append(examples, exampleFromQuery(q))
		}
	}
	return NewCorpus(examples), nil
}

// NewCorpus builds a corpus from in-memory examples (used in tests).
func NewCorpus(examples []Example) *Corpus {
	byUID := make(map[string]Example, len(examples))
	docs := make([]textrank.Document, 0, len(examples))
	for _, ex := range examples {
		byUID[ex.UID] = ex
		docs = append(docs, textrank.Document{
			ID: ex.UID,
			Parts: []textrank.WeightedText{
				{Text: ex.Title, Weight: weightTitle},
				{Text: ex.Desc, Weight: weightDesc},
				{Text: ex.Mql, Weight: weightMql},
			},
		})
	}
	return &Corpus{byUID: byUID, index: textrank.Build(docs)}
}

// Size reports the number of examples in the corpus.
func (c *Corpus) Size() int { return c.index.Len() }

// Search returns up to limit examples most similar to the intent, ranked by
// BM25 with a strong bonus for a matching provider.
func (c *Corpus) Search(intent, provider string, limit int) []Example {
	scored := c.index.SearchBonus(intent, limit, func(id string) float64 {
		if provider != "" && c.byUID[id].Provider == provider {
			return providerBonus
		}
		return 0
	})
	out := make([]Example, 0, len(scored))
	for _, s := range scored {
		ex := c.byUID[s.ID]
		ex.Score = s.Score
		out = append(out, ex)
	}
	return out
}

func exampleFromQuery(q *policy.Mquery) Example {
	desc := ""
	if q.Docs != nil {
		desc = q.Docs.Desc
	}
	if desc == "" {
		desc = q.Desc
	}
	provider, _ := ResolveProvider(Check{
		UID:     q.Uid,
		Filters: filterExpressions(q),
	})
	return Example{
		UID:      q.Uid,
		Title:    q.Title,
		Desc:     desc,
		Mql:      q.Mql,
		Provider: provider,
	}
}

// allBundleQueries enumerates every query definition in a bundle: top-level
// shared queries, plus checks and data queries nested in policy and pack groups.
func allBundleQueries(b *policy.Bundle) []*policy.Mquery {
	var out []*policy.Mquery
	out = append(out, b.Queries...)
	for _, p := range b.Policies {
		for _, g := range p.Groups {
			out = append(out, g.Checks...)
			out = append(out, g.Queries...)
		}
	}
	for _, pack := range b.Packs {
		out = append(out, pack.Queries...)
		for _, g := range pack.Groups {
			out = append(out, g.Queries...)
		}
	}
	return out
}

// filterExpressions returns the filter MQL strings attached to a query.
func filterExpressions(q *policy.Mquery) []string {
	if q.Filters == nil {
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
