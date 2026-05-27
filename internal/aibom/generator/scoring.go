// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generator

import (
	"go.mondoo.com/cnspec/v13/internal/aibom"
)

// Section weights for completeness scoring (ADR-0003).
var sectionWeights = map[string]float32{
	"identity":      0.25,
	"license_legal": 0.20,
	"technical":     0.20,
	"training_data": 0.20,
	"ethics_risk":   0.15,
}

// ComputeCompleteness calculates the completeness score for an AIBOM.
func ComputeCompleteness(models []*aibom.ModelComponent) *aibom.CompletenessScore {
	if len(models) == 0 {
		return &aibom.CompletenessScore{
			TotalScore:      0,
			SectionScores:   map[string]float32{},
			MissingFields:   []string{},
			Recommendations: []string{"No models discovered. Verify provider credentials and connectivity."},
		}
	}

	totalIdentity := float32(0)
	totalLicense := float32(0)
	totalTechnical := float32(0)
	totalTraining := float32(0)
	totalEthics := float32(0)

	var allMissing []string

	for _, m := range models {
		identity, missing := scoreIdentity(m)
		totalIdentity += identity
		allMissing = append(allMissing, missing...)

		license, missing := scoreLicense(m)
		totalLicense += license
		allMissing = append(allMissing, missing...)

		technical, missing := scoreTechnical(m)
		totalTechnical += technical
		allMissing = append(allMissing, missing...)

		training, missing := scoreTraining(m)
		totalTraining += training
		allMissing = append(allMissing, missing...)

		ethics, missing := scoreEthics(m)
		totalEthics += ethics
		allMissing = append(allMissing, missing...)
	}

	n := float32(len(models))
	sections := map[string]float32{
		"identity":      totalIdentity / n,
		"license_legal": totalLicense / n,
		"technical":     totalTechnical / n,
		"training_data": totalTraining / n,
		"ethics_risk":   totalEthics / n,
	}

	total := float32(0)
	for section, score := range sections {
		total += score * sectionWeights[section]
	}

	missing := deduplicate(allMissing)
	recs := generateRecommendations(sections)

	return &aibom.CompletenessScore{
		TotalScore:      total,
		SectionScores:   sections,
		MissingFields:   missing,
		Recommendations: recs,
	}
}

func scoreIdentity(m *aibom.ModelComponent) (float32, []string) {
	score := float32(0)
	total := float32(5)
	var missing []string

	if m.Name != "" {
		score++
	} else {
		missing = append(missing, m.Provider+":name")
	}
	if m.Version != "" {
		score++
	} else {
		missing = append(missing, m.Provider+":version")
	}
	if m.Provider != "" {
		score++
	}
	if m.ModelId != "" {
		score++
	} else {
		missing = append(missing, m.Provider+":model_id")
	}
	if m.Author != "" {
		score++
	} else {
		missing = append(missing, m.Provider+":author")
	}
	return score / total, missing
}

func scoreLicense(m *aibom.ModelComponent) (float32, []string) {
	if m.License != "" {
		return 1.0, nil
	}
	return 0, []string{m.Provider + ":license"}
}

func scoreTechnical(m *aibom.ModelComponent) (float32, []string) {
	score := float32(0)
	total := float32(4)
	var missing []string

	if m.Task != "" {
		score++
	} else {
		missing = append(missing, m.Provider+":task")
	}
	if m.ArchitectureFamily != "" || m.ModelArchitecture != "" {
		score++
	} else {
		missing = append(missing, m.Provider+":architecture")
	}
	if m.ApproachType != "" {
		score++
	} else {
		missing = append(missing, m.Provider+":approach_type")
	}
	if len(m.InputModalities) > 0 || len(m.OutputModalities) > 0 {
		score++
	} else {
		missing = append(missing, m.Provider+":modalities")
	}
	return score / total, missing
}

func scoreTraining(m *aibom.ModelComponent) (float32, []string) {
	score := float32(0)
	total := float32(2)
	var missing []string

	if len(m.TrainingDatasets) > 0 {
		score++
	} else {
		missing = append(missing, m.Provider+":training_datasets")
	}
	if len(m.Provenance) > 0 {
		score++
	} else {
		missing = append(missing, m.Provider+":provenance")
	}
	return score / total, missing
}

func scoreEthics(m *aibom.ModelComponent) (float32, []string) {
	score := float32(0)
	total := float32(3)
	var missing []string

	if len(m.EthicalConsiderations) > 0 {
		score++
	} else {
		missing = append(missing, m.Provider+":ethical_considerations")
	}
	if len(m.Limitations) > 0 {
		score++
	} else {
		missing = append(missing, m.Provider+":limitations")
	}
	if len(m.IntendedUses) > 0 {
		score++
	} else {
		missing = append(missing, m.Provider+":intended_uses")
	}
	return score / total, missing
}

func deduplicate(items []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

func generateRecommendations(sections map[string]float32) []string {
	var recs []string
	if sections["license_legal"] < 0.5 {
		recs = append(recs, "Most models are missing license information. Add license metadata to improve compliance readiness.")
	}
	if sections["training_data"] < 0.3 {
		recs = append(recs, "Training dataset provenance is largely unknown. Consider using providers with richer metadata (e.g., HuggingFace).")
	}
	if sections["ethics_risk"] < 0.2 {
		recs = append(recs, "Ethical considerations and intended use documentation is sparse. This is common for cloud-hosted models.")
	}
	if sections["technical"] < 0.3 {
		recs = append(recs, "Technical metadata (task, architecture, modalities) is incomplete for many models.")
	}
	return recs
}
