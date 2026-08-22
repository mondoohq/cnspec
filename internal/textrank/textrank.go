// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package textrank provides a small, dependency-free BM25 text ranker used to
// find relevant policy checks by meaning (as opposed to the identifier/substring
// matching in the policy graph's NodeIndex). It backs both `cnspec policy-graph
// search --similar` and the grounding search inside `cnspec policy generate`.
package textrank

import (
	"math"
	"sort"
)

// BM25 parameters. k1 controls term-frequency saturation, b controls length
// normalization. These are standard, robust defaults.
const (
	bm25K1 = 1.5
	bm25B  = 0.75
)

// WeightedText is a piece of a document's text and how strongly it counts. A
// higher weight makes matches in that field (e.g. a title) rank higher.
type WeightedText struct {
	Text   string
	Weight int
}

// Document is one rankable item, identified by ID, composed of weighted fields.
type Document struct {
	ID    string
	Parts []WeightedText
}

// Scored is a document ID and its relevance score.
type Scored struct {
	ID    string
	Score float64
}

// Index is a prebuilt BM25 index. Build it once and Search it many times.
type Index struct {
	ids       []string
	tf        []map[string]int // field-weighted term frequencies per document
	docLen    []int            // total weighted length per document
	idf       map[string]float64
	avgDocLen float64
}

// Build constructs an index from documents, computing field-weighted term
// frequencies and IDF weights up front.
func Build(docs []Document) *Index {
	ix := &Index{
		ids:    make([]string, len(docs)),
		tf:     make([]map[string]int, len(docs)),
		docLen: make([]int, len(docs)),
	}

	var totalLen int
	for i, d := range docs {
		ix.ids[i] = d.ID
		tf := map[string]int{}
		for _, p := range d.Parts {
			w := p.Weight
			if w <= 0 {
				w = 1
			}
			for _, t := range tokenize(p.Text) {
				tf[t] += w
			}
		}
		ix.tf[i] = tf
		for _, n := range tf {
			ix.docLen[i] += n
		}
		totalLen += ix.docLen[i]
	}

	n := float64(len(docs))
	if n == 0 {
		ix.idf = map[string]float64{}
		ix.avgDocLen = 1
		return ix
	}
	ix.avgDocLen = float64(totalLen) / n
	if ix.avgDocLen == 0 {
		ix.avgDocLen = 1
	}

	df := map[string]int{}
	for _, tf := range ix.tf {
		for t := range tf {
			df[t]++
		}
	}
	ix.idf = make(map[string]float64, len(df))
	for t, d := range df {
		// BM25 idf with +1 so it is always positive, even for very common terms
		ix.idf[t] = math.Log(1 + (n-float64(d)+0.5)/(float64(d)+0.5))
	}
	return ix
}

// Len reports the number of indexed documents.
func (ix *Index) Len() int { return len(ix.ids) }

// Search returns up to limit documents ranked by BM25 relevance to query.
func (ix *Index) Search(query string, limit int) []Scored {
	return ix.SearchBonus(query, limit, nil)
}

// SearchBonus ranks documents by BM25 plus an optional per-document bonus (e.g.
// a provider match), applied before sorting. A nil bonus is a plain BM25 search.
func (ix *Index) SearchBonus(query string, limit int, bonus func(id string) float64) []Scored {
	if limit <= 0 {
		limit = 10
	}
	terms := dedupe(tokenize(query))
	if len(terms) == 0 {
		return nil
	}

	scored := make([]Scored, 0, len(ix.ids))
	for i, id := range ix.ids {
		score := ix.bm25(terms, i)
		if bonus != nil {
			score += bonus(id)
		}
		if score <= 0 {
			continue
		}
		scored = append(scored, Scored{ID: id, Score: score})
	}

	sort.SliceStable(scored, func(a, b int) bool {
		if scored[a].Score != scored[b].Score {
			return scored[a].Score > scored[b].Score
		}
		return scored[a].ID < scored[b].ID
	})

	if len(scored) > limit {
		scored = scored[:limit]
	}
	return scored
}

func (ix *Index) bm25(terms []string, i int) float64 {
	tf := ix.tf[i]
	dl := float64(ix.docLen[i])
	var score float64
	for _, t := range terms {
		f := float64(tf[t])
		if f == 0 {
			continue
		}
		norm := f * (bm25K1 + 1) / (f + bm25K1*(1-bm25B+bm25B*dl/ix.avgDocLen))
		score += ix.idf[t] * norm
	}
	return score
}

func dedupe(tokens []string) []string {
	seen := map[string]bool{}
	out := tokens[:0:0]
	for _, t := range tokens {
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}
