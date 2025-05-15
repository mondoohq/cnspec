// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"encoding/json"
	"reflect"

	"github.com/rs/zerolog/log"
	cnquery_reporter "go.mondoo.com/cnquery/v11/cli/reporter"
)

func FindSameAsset(name string, assets map[string]*cnquery_reporter.Asset) string {
	for k := range assets {
		if assets[k].Name == name {
			return assets[k].Mrn
		}
	}
	return ""
}

func CompareReports(baseReport, compareReport *Report) bool {
	equal := true
	for mrn := range baseReport.Assets {

		baseAsset := baseReport.Assets[mrn]
		// best bet is to compare the asset based on the name until we have the platform id exposed
		similarAsset := FindSameAsset(baseAsset.Name, compareReport.Assets)
		if similarAsset == "" {
			log.Info().Msgf("🔴 asset %q is missing in compare report", baseAsset.Name)
			equal = true
			continue
		}

		cmp := CompareAsset(baseReport, mrn, compareReport, similarAsset)
		if !cmp {
			equal = false
		}
	}

	if equal {
		log.Info().Msg("🔴 reports differ")
	} else {
		log.Info().Msg("✅ reports are equal")
	}
	return equal
}

// CompareAsset returns true if the reports are equal
func CompareAsset(baseReport *Report, baseAssetMrn string, compareReport *Report, compareAssetMrn string) bool {
	log.Info().Msgf("🔍 comparing asset %s with %s", baseReport.Assets[baseAssetMrn].Name, compareReport.Assets[compareAssetMrn].Name)

	equal := true
	// compare asset data, we ignore the mrn field
	baseAsset := baseReport.Assets[baseAssetMrn]
	baseAsset.Mrn = ""
	compareAsset := compareReport.Assets[compareAssetMrn]
	compareAsset.Mrn = ""
	if !reflect.DeepEqual(baseAsset, compareAsset) {
		log.Info().Msgf("🔴 assets are different:")
		log.Info().Msgf("   expected: %s", printAsJSON(baseAsset))
		log.Info().Msgf("   got     : %s", printAsJSON(compareAsset))
		equal = false
	}

	// gather scores
	baseAssetScores := baseReport.Scores[baseAssetMrn]
	compareAssetScores := compareReport.Scores[compareAssetMrn]

	// compare asset scores
	if !compareScores(baseAssetMrn, baseAssetScores.GetScore(baseAssetMrn), compareAssetScores.GetScore(compareAssetMrn)) {
		equal = false
	}

	// compare checks results
	visited := make(map[string]bool)
	for check, baseResult := range baseAssetScores.Values {
		// ignore base asset mrn
		if check == baseAssetMrn {
			continue
		}

		compareResult := compareAssetScores.GetScore(check)
		if compareResult == nil {
			log.Info().Msgf("🔴 check %q is missing in compare report", check)
			equal = false
			continue
		}

		if !compareScores(check, baseResult, compareResult) {
			equal = false
		}

		visited[check] = true
	}

	// check if there are any checks in the compare report that are not in the base report
	for checkMrn := range compareAssetScores.Values {
		if checkMrn == baseAssetMrn || checkMrn == compareAssetMrn {
			continue
		}
		if !visited[checkMrn] {
			log.Info().Msgf("🔴 check %q is missing in base report", checkMrn)
			equal = false
		}
	}

	return equal
}

func printAsJSON(v any) string {
	data, _ := json.Marshal(v)
	return string(data)
}

// compareScores compares the scores of two checks, if they are not equal, it will log the difference
// and return false, otherwise it will return true
func compareScores(check string, baseResults *ScoreValue, compareResults *ScoreValue) bool {
	if !reflect.DeepEqual(baseResults, compareResults) {
		log.Info().Msgf("🔴 check %q got different results", check)
		log.Info().Msgf("   expected:      %s", printAsJSON(baseResults))
		log.Info().Msgf("   got     : %s", printAsJSON(compareResults))
		return false
	}
	return true
}
