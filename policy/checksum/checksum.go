// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package checksum computes the scan-content checksums for the row types
// that live in cnspec — scores and scored risk factors. It builds on mql's
// llx/checksum, which owns the hash writer, the llx canonicalization, and
// the data/resource row checksums; this package adds exactly what cannot
// live there without a dependency cycle (mql cannot import cnspec's policy
// protos). The server imports both packages and re-implements neither.
//
// The canonicalization rules (explicit canon, no reflection, multiset
// lists, length-prefixed writes) and the AlgoVersion are llx/checksum's;
// see that package's doc.
package checksum

import (
	"go.mondoo.com/cnspec/policy"
	llxchecksum "go.mondoo.com/mql/llx/checksum"
)

// canonScoredRiskFactor hashes one ScoredRiskFactor message, in full —
// unlike HashRiskRow, which hashes only the four scandb table columns.
func canonScoredRiskFactor(h *llxchecksum.Hasher, r *policy.ScoredRiskFactor) error {
	if r == nil {
		h.Str("<nil>")
		return nil
	}
	h.Str(r.Mrn).F64(float64(r.Risk)).Bool(r.IsToxic).Bool(r.IsDetected)
	return llxchecksum.CanonResultMap(h, r.Data)
}

// canonScoredRiskFactors folds on emptiness, not pointer presence: the
// scandb round trip marshals an empty container to zero bytes and reads it
// back as nil (writeScore / convertScore), so nil and empty MUST hash
// identically or the same logical row would checksum differently before
// and after storage. Empty collapses to the nil encoding — the form every
// stored row round-trips to — so hashing an in-memory row and hashing its
// stored self always agree.
func canonScoredRiskFactors(h *llxchecksum.Hasher, rf *policy.ScoredRiskFactors) error {
	if rf == nil || len(rf.Items) == 0 {
		h.Str("<nil>")
		return nil
	}
	h.Str("risk_factors")
	return llxchecksum.Multiset(h, "risk_factor", rf.Items, canonScoredRiskFactor)
}

// canonSource hashes a score source's identity, scanner fields, and fixed
// state. FirstDetectedAt and LastUpdatedAt are wall-clock noise and
// deliberately excluded — the same rationale as HashResourceRow dropping
// Created/Updated: a re-scan that changes nothing but observation times
// must not read as changed content. FixedAt is NOT observation noise: it
// flips once, when the source marks the finding fixed, and a scan whose
// only difference is that flip must read as changed so the server learns
// the fixed state instead of keeping a stale un-fixed copy.
func canonSource(h *llxchecksum.Hasher, s *policy.Source) error {
	if s == nil {
		h.Str("<nil>")
		return nil
	}
	h.Str(s.Name).Str(s.Url).Str(s.Version).I64(int64(s.Vendor)).Str(s.FixedAt)
	return nil
}

// canonSources folds on emptiness for the same reason as
// canonScoredRiskFactors: nil and empty are one value after the storage
// round trip, so they must be one value in the hash.
func canonSources(h *llxchecksum.Hasher, s *policy.Sources) error {
	if s == nil || len(s.Items) == 0 {
		h.Str("<nil>")
		return nil
	}
	h.Str("sources")
	return llxchecksum.Multiset(h, "source", s.Items, canonSource)
}

// HashScoreRow computes the row checksum for a scores row.
func HashScoreRow(s *policy.Score) (uint64, error) {
	h := llxchecksum.NewHasher("row:score")
	if s == nil {
		h.Str("<nil>")
		if err := h.Err(); err != nil {
			return 0, err
		}
		return h.Sum64(), nil
	}
	h.Str(s.QrId).U32(s.RiskScore).U32(s.Type).U32(s.Value).U32(s.Weight).Str(s.Message)
	if err := canonScoredRiskFactors(h, s.RiskFactors); err != nil {
		return 0, err
	}
	if err := canonSources(h, s.Sources); err != nil {
		return 0, err
	}
	if err := h.Err(); err != nil {
		return 0, err
	}
	return h.Sum64(), nil
}

// HashRiskRow computes the row checksum for a scored_risk_factors row. Only
// the four table columns are content here — the Data map is not stored in
// the table.
//
// Float canonicalization: the canonical width is the proto field's float32.
// The risk column is REAL (float64) on disk, so every hasher must narrow
// the column value to float32 and re-widen before folding — cnspec's
// readers do exactly that (StreamRisks/GetRisk build a float32
// ScoredRiskFactor, and this function folds float64(float32)). A server
// recomputing from the column must narrow the same way; hashing the raw
// REAL diverges permanently for any stored value that is not exactly
// float32-representable.
func HashRiskRow(r *policy.ScoredRiskFactor) (uint64, error) {
	h := llxchecksum.NewHasher("row:risk")
	if r == nil {
		h.Str("<nil>")
		if err := h.Err(); err != nil {
			return 0, err
		}
		return h.Sum64(), nil
	}
	h.Str(r.Mrn).F64(float64(r.Risk)).Bool(r.IsToxic).Bool(r.IsDetected)
	if err := h.Err(); err != nil {
		return 0, err
	}
	return h.Sum64(), nil
}
