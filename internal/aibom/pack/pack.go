// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package pack

import (
	_ "embed"

	"go.mondoo.com/cnspec/v13/policy"
)

//go:embed aibom.mql.yaml
var aibomQueryPack []byte

// QueryPack returns the AIBOM query pack as a policy bundle.
// The bundle contains queries for collecting AI model inventory
// information from providers including Ollama, HuggingFace,
// AWS Bedrock/SageMaker, GCP Vertex AI, and Azure AI Services.
func QueryPack() (*policy.Bundle, error) {
	bundle, err := policy.BundleFromYAML(aibomQueryPack)
	if err != nil {
		return nil, err
	}
	bundle.ConvertQuerypacks()
	return bundle, nil
}
