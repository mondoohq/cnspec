// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
)

func Query2Assessment(bundle *llx.CodeBundle, report *Report) *llx.Assessment {
	if bundle.GetCodeV2() == nil {
		return nil
	}

	if report == nil {
		report = &Report{}
	}

	// TODO: investigate why this is ever nil but it is sometimes
	if report.Data == nil {
		report.Data = map[string]*llx.Result{}
	}

	// TODO: we might want to store these differently per-entrypoint
	if score, ok := report.Scores[bundle.CodeV2.Id]; ok {
		entrypoints := bundle.CodeV2.Entrypoints()
		if len(entrypoints) == 1 {
			c := bundle.CodeV2.Checksums[entrypoints[0]]
			if _, ok := report.Data[c]; !ok {
				if score.Value == 100 {
					report.Data[c] = llx.BoolTrue.Result()
				} else {
					report.Data[c] = llx.BoolFalse.Result()
				}
			}
		}
	}

	return llx.Results2AssessmentLookupV2(bundle, func(s string) (*llx.RawResult, bool) {
		score, ok := report.Scores[s]
		if ok {
			if score.Value == 100 {
				return &llx.RawResult{
					CodeID: s,
					Data:   llx.BoolTrue,
				}, true
			}

			return &llx.RawResult{
				CodeID: s,
				Data:   llx.BoolFalse,
			}, true
		}

		data, ok := report.Data[s]
		if ok && data != nil {
			return data.RawResultV2(), true
		}

		log.Debug().
			Str("codeID", bundle.CodeV2.Id).
			Str("checksum", s).
			Bool("found-but-nil", ok).
			Msg("could not look up result for field in query")
		return nil, false
	})
}
