// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package textrank

import "testing"

func doc(id string, title, body string) Document {
	return Document{ID: id, Parts: []WeightedText{
		{Text: title, Weight: 3},
		{Text: body, Weight: 1},
	}}
}

func TestBM25Ranking(t *testing.T) {
	ix := Build([]Document{
		doc("s3", "S3 buckets must be encrypted", "aws.s3.buckets.all(encryption != empty)"),
		doc("disk", "Compute disks must be encrypted", "gcp.compute.disks.all(diskEncryptionKey != empty)"),
		doc("ssh", "SSH root login disabled", `sshd.config.params["PermitRootLogin"] == "no"`),
	})

	got := ix.Search("bucket encryption", 2)
	if len(got) == 0 || got[0].ID != "s3" {
		t.Fatalf("expected s3 first, got %+v", got)
	}
}

func TestStemmingMatch(t *testing.T) {
	ix := Build([]Document{
		doc("a", "S3 buckets must be encrypted", ""),
		doc("b", "SSH login policies", ""),
	})
	// singular query should match plural indexed term via stemming
	got := ix.Search("bucket encrypt", 1)
	if len(got) != 1 || got[0].ID != "a" {
		t.Fatalf("expected stemmed match a, got %+v", got)
	}
}

func TestSearchBonus(t *testing.T) {
	ix := Build([]Document{
		doc("aws", "storage encryption", "aws.s3"),
		doc("gcp", "storage encryption", "gcp.storage"),
	})
	// identical text; the bonus must break the tie deterministically
	got := ix.SearchBonus("storage encryption", 2, func(id string) float64 {
		if id == "gcp" {
			return 5
		}
		return 0
	})
	if len(got) != 2 || got[0].ID != "gcp" {
		t.Fatalf("expected gcp first via bonus, got %+v", got)
	}
}

func TestEmptyIndexAndQuery(t *testing.T) {
	if got := Build(nil).Search("anything", 5); len(got) != 0 {
		t.Fatalf("empty index should return no results, got %+v", got)
	}
	ix := Build([]Document{doc("a", "hello world", "")})
	if got := ix.Search("", 5); len(got) != 0 {
		t.Fatalf("empty query should return no results, got %+v", got)
	}
	// query of only stopwords tokenizes to nothing
	if got := ix.Search("the a of", 5); len(got) != 0 {
		t.Fatalf("stopword-only query should return no results, got %+v", got)
	}
}
