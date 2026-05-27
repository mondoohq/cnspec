// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generator

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnspec/v13/internal/aibom"
)

// whisperTinyHuggingFace returns a HuggingFaceModel that simulates the MQL
// report data for openai/whisper-tiny, matching the real HuggingFace API response.
func whisperTinyHuggingFace() HuggingFaceModel {
	return HuggingFaceModel{
		ID:          "openai/whisper-tiny",
		ModelID:     "openai/whisper-tiny",
		Author:      "openai",
		PipelineTag: "automatic-speech-recognition",
		LibraryName: "transformers",
		Tags: []string{
			"transformers", "pytorch", "tf", "jax", "safetensors",
			"whisper", "automatic-speech-recognition",
			"audio", "hf-agentic-benchmark",
			"en", "zh", "de", "es", "ru", "ko", "fr", "ja", "pt", "tr",
			"arxiv:2212.04356",
			"license:apache-2.0",
		},
		Downloads:    12345678,
		Likes:        5678,
		SHA:          "169d4a4341b33bc18d8881c4b69c2e104e1cc0af",
		CreatedAt:    "2022-09-26T07:54:22.000Z",
		LastModified: "2024-12-17T16:45:32.000Z",
		License:      "apache-2.0",
		CardData: map[string]any{
			"language":     []any{"en", "zh", "de", "es", "ru", "ko", "fr", "ja", "pt", "tr"},
			"tags":         []any{"audio", "automatic-speech-recognition"},
			"pipeline_tag": "automatic-speech-recognition",
			"license":      "apache-2.0",
			"datasets":     []any{"680k", "name"},
		},
		Config: map[string]any{
			"architectures":   []any{"WhisperForConditionalGeneration"},
			"model_type":      "whisper",
			"vocab_size":      float64(51865),
			"tokenizer_class": "WhisperTokenizer",
		},
	}
}

func TestHuggingFaceWhisperTiny_ModelComponent(t *testing.T) {
	hf := whisperTinyHuggingFace()
	mc := huggingfaceToModelComponent(hf)

	// Name should be the short name, not the full ID
	assert.Equal(t, "whisper-tiny", mc.Name)
	assert.Equal(t, "openai", mc.Author)
	assert.Equal(t, "huggingface", mc.Provider)

	// Version should be first 8 chars of SHA (OWASP AIBOM standard)
	assert.Equal(t, "169d4a43", mc.Version)

	// PURL should include namespace (author)
	assert.Equal(t, "pkg:huggingface/openai/whisper-tiny@169d4a43", mc.Purl)

	// Task from pipelineTag
	assert.Equal(t, "automatic-speech-recognition", mc.Task)

	// Architecture from config
	assert.Equal(t, "whisper", mc.ArchitectureFamily)
	assert.Equal(t, "WhisperForConditionalGeneration", mc.ModelArchitecture)

	// License
	assert.Equal(t, "apache-2.0", mc.License)

	// Datasets from cardData
	assert.Equal(t, []string{"680k", "name"}, mc.TrainingDatasets)

	// Input/output modalities derived from pipelineTag
	assert.Equal(t, []string{"audio"}, mc.InputModalities)
	assert.Equal(t, []string{"text"}, mc.OutputModalities)

	// Config-derived labels
	assert.Equal(t, "51865", mc.Labels["vocab_size"])
	assert.Equal(t, "WhisperTokenizer", mc.Labels["tokenizer_class"])

	// Source URL
	assert.Equal(t, "https://huggingface.co/openai/whisper-tiny", mc.SourceUrl)

	// Provenance should contain full SHA
	assert.Equal(t, "169d4a4341b33bc18d8881c4b69c2e104e1cc0af", mc.Provenance["sha"])
	assert.Equal(t, "transformers", mc.Provenance["library"])
	assert.Equal(t, "openai/whisper-tiny", mc.Provenance["full_name"])

	// ArXiv should be extracted from tags
	assert.Equal(t, "2212.04356", mc.Provenance["arxiv"])
}

func TestHuggingFaceWhisperTiny_EmptyAuthor(t *testing.T) {
	hf := whisperTinyHuggingFace()
	hf.Author = "" // Simulates list API behavior (no author field)
	hf.SHA = ""    // List API also omits SHA

	mc := huggingfaceToModelComponent(hf)

	// Author should be extracted from the model ID
	assert.Equal(t, "openai", mc.Author)
	assert.Equal(t, "whisper-tiny", mc.Name)

	// Without SHA, version should be empty and PURL should have trailing @
	assert.Equal(t, "", mc.Version)
	assert.Equal(t, "pkg:huggingface/openai/whisper-tiny@", mc.Purl)
}

func TestHuggingFaceWhisperTiny_CycloneDX(t *testing.T) {
	hf := whisperTinyHuggingFace()
	mc := huggingfaceToModelComponent(hf)

	bom := &aibom.AiBom{
		Generator: &aibom.Generator{
			Vendor:  "Mondoo, Inc.",
			Name:    "cnspec",
			Version: "11.0.0",
		},
		Asset: &aibom.Asset{
			Name: "openai/whisper-tiny",
		},
		Models:       []*aibom.ModelComponent{mc},
		Completeness: &aibom.CompletenessScore{TotalScore: 0.646},
	}

	// Render to CycloneDX JSON
	formatter := &aibom.CycloneDXFormatter{Format: 1} // JSON format
	var buf bytes.Buffer
	err := formatter.Render(&buf, bom)
	require.NoError(t, err)

	// Parse the output
	var cdx map[string]any
	err = json.Unmarshal(buf.Bytes(), &cdx)
	require.NoError(t, err)

	// Verify top-level structure
	assert.Equal(t, "CycloneDX", cdx["bomFormat"])
	assert.NotEmpty(t, cdx["serialNumber"])
	assert.NotNil(t, cdx["metadata"])
	assert.NotNil(t, cdx["components"])

	// Get the first component
	components := cdx["components"].([]any)
	require.Len(t, components, 1)
	comp := components[0].(map[string]any)

	// Verify component fields match OWASP reference structure
	assert.Equal(t, "machine-learning-model", comp["type"])
	assert.Equal(t, "openai", comp["group"])
	assert.Equal(t, "whisper-tiny", comp["name"])
	assert.Equal(t, "169d4a43", comp["version"])
	assert.Equal(t, "pkg:huggingface/openai/whisper-tiny@169d4a43", comp["purl"])

	// Verify bom-ref uses PURL
	assert.Equal(t, "pkg:huggingface/openai/whisper-tiny@169d4a43", comp["bom-ref"])

	// Verify license uses SPDX ID
	licenses := comp["licenses"].([]any)
	require.Len(t, licenses, 1)
	licenseEntry := licenses[0].(map[string]any)
	licenseObj := licenseEntry["license"].(map[string]any)
	assert.Equal(t, "Apache-2.0", licenseObj["id"])

	// Verify supplier and manufacturer
	supplier := comp["supplier"].(map[string]any)
	assert.Equal(t, "openai", supplier["name"])

	manufacturer := comp["manufacturer"].(map[string]any)
	assert.Equal(t, "openai", manufacturer["name"])

	// Verify authors
	authors := comp["authors"].([]any)
	require.Len(t, authors, 1)
	assert.Equal(t, "openai", authors[0].(map[string]any)["name"])

	// Verify modelCard
	modelCard := comp["modelCard"].(map[string]any)
	require.NotNil(t, modelCard)

	modelParams := modelCard["modelParameters"].(map[string]any)
	assert.Equal(t, "automatic-speech-recognition", modelParams["task"])
	assert.Equal(t, "whisper", modelParams["architectureFamily"])

	// Verify datasets
	datasets := modelParams["datasets"].([]any)
	require.Len(t, datasets, 2)
	ds0 := datasets[0].(map[string]any)
	assert.Equal(t, "dataset", ds0["type"])
	assert.Equal(t, "680k", ds0["name"])

	// Verify inputs/outputs
	inputs := modelParams["inputs"].([]any)
	require.Len(t, inputs, 1)
	assert.Equal(t, "audio", inputs[0].(map[string]any)["format"])

	outputs := modelParams["outputs"].([]any)
	require.Len(t, outputs, 1)
	assert.Equal(t, "text", outputs[0].(map[string]any)["format"])

	// Verify external references
	extRefs := comp["externalReferences"].([]any)
	require.GreaterOrEqual(t, len(extRefs), 4)

	// Should have website, distribution, VCS, and documentation references
	refTypes := map[string]bool{}
	for _, ref := range extRefs {
		r := ref.(map[string]any)
		refTypes[r["type"].(string)] = true
	}
	assert.True(t, refTypes["website"], "should have website reference")
	assert.True(t, refTypes["distribution"], "should have distribution reference")
	assert.True(t, refTypes["vcs"], "should have VCS reference")
	assert.True(t, refTypes["documentation"], "should have documentation reference")
}

func TestHuggingFaceWhisperTiny_CycloneDX_MatchesOWASPStructure(t *testing.T) {
	hf := whisperTinyHuggingFace()
	mc := huggingfaceToModelComponent(hf)

	bom := &aibom.AiBom{
		Generator: &aibom.Generator{
			Vendor:  "Mondoo, Inc.",
			Name:    "cnspec",
			Version: "11.0.0",
		},
		Asset: &aibom.Asset{
			Name: "openai/whisper-tiny",
		},
		Models:       []*aibom.ModelComponent{mc},
		Completeness: &aibom.CompletenessScore{TotalScore: 0.646},
	}

	formatter := &aibom.CycloneDXFormatter{Format: 1} // JSON
	var buf bytes.Buffer
	err := formatter.Render(&buf, bom)
	require.NoError(t, err)

	var cdx map[string]any
	err = json.Unmarshal(buf.Bytes(), &cdx)
	require.NoError(t, err)

	comp := cdx["components"].([]any)[0].(map[string]any)

	// Verify VCS reference includes full commit SHA
	extRefs := comp["externalReferences"].([]any)
	var vcsRef map[string]any
	for _, ref := range extRefs {
		r := ref.(map[string]any)
		if r["type"] == "vcs" {
			vcsRef = r
			break
		}
	}
	require.NotNil(t, vcsRef, "should have a VCS external reference")
	assert.Equal(t,
		"https://huggingface.co/openai/whisper-tiny/commit/169d4a4341b33bc18d8881c4b69c2e104e1cc0af",
		vcsRef["url"],
	)
	assert.Equal(t, "Specific commit", vcsRef["comment"])

	// Verify ArXiv documentation reference
	var docRef map[string]any
	for _, ref := range extRefs {
		r := ref.(map[string]any)
		if r["type"] == "documentation" {
			docRef = r
			break
		}
	}
	require.NotNil(t, docRef, "should have a documentation reference for ArXiv paper")
	assert.Equal(t, "https://arxiv.org/abs/2212.04356", docRef["url"])
	assert.Equal(t, "ArXiv Paper", docRef["comment"])

	// Verify training dataset external references
	var datasetRefs []map[string]any
	for _, ref := range extRefs {
		r := ref.(map[string]any)
		comment, _ := r["comment"].(string)
		if r["type"] == "distribution" && len(comment) > 0 && comment != "Model files" {
			datasetRefs = append(datasetRefs, r)
		}
	}
	require.Len(t, datasetRefs, 2)
	assert.Equal(t, "https://huggingface.co/datasets/680k", datasetRefs[0]["url"])
	assert.Equal(t, "Training dataset: 680k", datasetRefs[0]["comment"])

	// Verify properties include vocab_size and tokenizer_class
	props := comp["properties"].([]any)
	propMap := map[string]string{}
	for _, p := range props {
		prop := p.(map[string]any)
		propMap[prop["name"].(string)] = prop["value"].(string)
	}
	assert.Equal(t, "51865", propMap["genai:aibom:modelcard:vocab_size"])
	assert.Equal(t, "WhisperTokenizer", propMap["genai:aibom:modelcard:tokenizer_class"])
	assert.Equal(t, "huggingface", propMap["mondoo:model:provider"])
}

func TestLocalModelToComponent_HuggingFaceCache(t *testing.T) {
	m := LocalAIModel{
		Name:          "Meta-Llama-3-8B-Instruct",
		Source:        "huggingface",
		Vendor:        "meta-llama",
		Family:        "llama",
		Path:          "/home/user/.cache/huggingface/hub/models--meta-llama--Meta-Llama-3-8B-Instruct",
		Size:          16065438720,
		Format:        "safetensors",
		Version:       "abc123de",
		Architecture:  "LlamaForCausalLM",
		License:       "llama3",
		ParameterSize: "8B",
	}

	mc := localModelToComponent(m)

	assert.Equal(t, "Meta-Llama-3-8B-Instruct", mc.Name)
	assert.Equal(t, "huggingface", mc.Provider)
	assert.Equal(t, "meta-llama", mc.Author)
	assert.Equal(t, "llama", mc.ArchitectureFamily)
	assert.Equal(t, "LlamaForCausalLM", mc.ModelArchitecture)
	assert.Equal(t, "safetensors", mc.Format)
	assert.Equal(t, "abc123de", mc.Version)
	assert.Equal(t, "8B", mc.ParameterSize)
	assert.Equal(t, "llama3", mc.License)

	assert.Equal(t, "file:///home/user/.cache/huggingface/hub/models--meta-llama--Meta-Llama-3-8B-Instruct", mc.SourceUrl)
	assert.Equal(t, "16065438720", mc.Labels["size_bytes"])
	assert.Equal(t, "ai.models", mc.Provenance["detection_source"])
	assert.Equal(t, m.Path, mc.Provenance["local_path"])

	assert.Equal(t, "pkg:huggingface/meta-llama/Meta-Llama-3-8B-Instruct@abc123de", mc.Purl)
}

func TestLocalModelToComponent_LMStudio(t *testing.T) {
	m := LocalAIModel{
		Name:         "mistral-7b-instruct-v0.2.Q4_K_M.gguf",
		Source:       "lmstudio",
		Vendor:       "TheBloke",
		Family:       "mistral",
		Path:         "/home/user/.cache/lm-studio/models/TheBloke/mistral-7b-instruct-v0.2.Q4_K_M.gguf",
		Format:       "gguf",
		Quantization: "Q4_K_M",
	}

	mc := localModelToComponent(m)

	assert.Equal(t, "mistral-7b-instruct-v0.2.Q4_K_M.gguf", mc.Name)
	assert.Equal(t, "lmstudio", mc.Provider)
	assert.Equal(t, "TheBloke", mc.Author)
	assert.Equal(t, "gguf", mc.Format)
	assert.Equal(t, "Q4_K_M", mc.Quantization)

	assert.Equal(t, "pkg:generic/lmstudio/TheBloke/mistral-7b-instruct-v0.2.Q4_K_M.gguf@", mc.Purl)
}

func TestLocalModelToComponent_MinimalFields(t *testing.T) {
	m := LocalAIModel{
		Name:   "resnet50",
		Source: "pytorch",
		Path:   "/home/user/.cache/torch/hub/checkpoints/resnet50.pth",
	}

	mc := localModelToComponent(m)

	assert.Equal(t, "resnet50", mc.Name)
	assert.Equal(t, "pytorch", mc.Provider)
	assert.Equal(t, "", mc.Author)
	assert.Equal(t, "", mc.Version)
	assert.Equal(t, "file:///home/user/.cache/torch/hub/checkpoints/resnet50.pth", mc.SourceUrl)
	assert.Equal(t, "ai.models", mc.Provenance["detection_source"])

	assert.Equal(t, "pkg:generic/pytorch/resnet50@", mc.Purl)
}

func TestLocalModelPurl(t *testing.T) {
	tests := []struct {
		name     string
		model    LocalAIModel
		expected string
	}{
		{
			name:     "ollama with vendor",
			model:    LocalAIModel{Name: "llama3", Source: "ollama", Vendor: "meta", Version: "latest"},
			expected: "pkg:ollama/meta/llama3@latest",
		},
		{
			name:     "ollama without vendor",
			model:    LocalAIModel{Name: "llama3", Source: "ollama", Version: "latest"},
			expected: "pkg:ollama/llama3@latest",
		},
		{
			name:     "huggingface",
			model:    LocalAIModel{Name: "bert-base", Source: "huggingface", Vendor: "google", Version: "abc123"},
			expected: "pkg:huggingface/google/bert-base@abc123",
		},
		{
			name:     "lmstudio",
			model:    LocalAIModel{Name: "model.gguf", Source: "lmstudio", Vendor: "TheBloke", Version: "v1"},
			expected: "pkg:generic/lmstudio/TheBloke/model.gguf@v1",
		},
		{
			name:     "gpt4all",
			model:    LocalAIModel{Name: "falcon", Source: "gpt4all", Vendor: "nomic"},
			expected: "pkg:generic/gpt4all/nomic/falcon@",
		},
		{
			name:     "pytorch no vendor",
			model:    LocalAIModel{Name: "resnet50", Source: "pytorch"},
			expected: "pkg:generic/pytorch/resnet50@",
		},
		{
			name:     "keras",
			model:    LocalAIModel{Name: "efficientnet", Source: "keras", Vendor: "google", Version: "b0"},
			expected: "pkg:generic/keras/google/efficientnet@b0",
		},
		{
			name:     "tfhub",
			model:    LocalAIModel{Name: "universal-sentence-encoder", Source: "tfhub", Vendor: "google"},
			expected: "pkg:generic/tfhub/google/universal-sentence-encoder@",
		},
		{
			name:     "jan",
			model:    LocalAIModel{Name: "tinyllama", Source: "jan", Vendor: "", Version: "1.1"},
			expected: "pkg:generic/jan/tinyllama@1.1",
		},
		{
			name:     "empty source",
			model:    LocalAIModel{Name: "model", Version: "1.0"},
			expected: "pkg:generic/generic/model@1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, localModelPurl(tt.model))
		})
	}
}

func TestLocalModel_CycloneDX(t *testing.T) {
	m := LocalAIModel{
		Name:          "phi-3-mini",
		Source:        "lmstudio",
		Vendor:        "microsoft",
		Family:        "phi",
		Path:          "/home/user/.cache/lm-studio/models/microsoft/phi-3-mini.gguf",
		Size:          4000000000,
		Format:        "gguf",
		Version:       "v3.8",
		Quantization:  "Q4_K_M",
		ParameterSize: "3.8B",
		License:       "MIT",
	}

	mc := localModelToComponent(m)

	bom := &aibom.AiBom{
		Generator: &aibom.Generator{
			Vendor:  "Mondoo, Inc.",
			Name:    "cnspec",
			Version: "11.0.0",
		},
		Asset: &aibom.Asset{
			Name: "local-workstation",
		},
		Models:       []*aibom.ModelComponent{mc},
		Completeness: &aibom.CompletenessScore{TotalScore: 0.5},
	}

	formatter := &aibom.CycloneDXFormatter{Format: 1}
	var buf bytes.Buffer
	err := formatter.Render(&buf, bom)
	require.NoError(t, err)

	var cdx map[string]any
	err = json.Unmarshal(buf.Bytes(), &cdx)
	require.NoError(t, err)

	components := cdx["components"].([]any)
	require.Len(t, components, 1)
	comp := components[0].(map[string]any)

	assert.Equal(t, "machine-learning-model", comp["type"])
	assert.Equal(t, "microsoft", comp["group"])
	assert.Equal(t, "phi-3-mini", comp["name"])
	assert.Equal(t, "v3.8", comp["version"])

	purl := "pkg:generic/lmstudio/microsoft/phi-3-mini@v3.8"
	assert.Equal(t, purl, comp["purl"])
	assert.Equal(t, purl, comp["bom-ref"])

	// Verify file:// URL does NOT generate a broken distribution reference
	extRefs := comp["externalReferences"].([]any)
	for _, ref := range extRefs {
		r := ref.(map[string]any)
		if r["type"] == "distribution" {
			t.Error("file:// URL should not generate a distribution reference with /tree/main")
		}
	}

	// Should have a website reference with the file:// URL
	var hasWebsite bool
	for _, ref := range extRefs {
		r := ref.(map[string]any)
		if r["type"] == "website" {
			assert.Equal(t, "file:///home/user/.cache/lm-studio/models/microsoft/phi-3-mini.gguf", r["url"])
			hasWebsite = true
		}
	}
	assert.True(t, hasWebsite, "should have website reference with file:// URL")

	// Verify properties include format, quantization, provider
	props := comp["properties"].([]any)
	propMap := map[string]string{}
	for _, p := range props {
		prop := p.(map[string]any)
		propMap[prop["name"].(string)] = prop["value"].(string)
	}
	assert.Equal(t, "lmstudio", propMap["mondoo:model:provider"])
	assert.Equal(t, "Q4_K_M", propMap["mondoo:model:quantization"])
	assert.Equal(t, "3.8B", propMap["mondoo:model:parameterSize"])
	assert.Equal(t, "gguf", propMap["mondoo:model:format"])
}

func TestClassifyAIDependencies(t *testing.T) {
	pythonPkgs := []SoftwarePackage{
		{Name: "transformers", Version: "4.40.0", Purl: "pkg:pypi/transformers@4.40.0"},
		{Name: "openai", Version: "1.30.0", Purl: "pkg:pypi/openai@1.30.0"},
		{Name: "langchain", Version: "0.2.0", Purl: "pkg:pypi/langchain@0.2.0"},
		{Name: "chromadb", Version: "0.5.0", Purl: "pkg:pypi/chromadb@0.5.0"},
		{Name: "wandb", Version: "0.17.0", Purl: "pkg:pypi/wandb@0.17.0"},
		{Name: "requests", Version: "2.32.0", Purl: "pkg:pypi/requests@2.32.0"},
		{Name: "flask", Version: "3.0.0", Purl: "pkg:pypi/flask@3.0.0"},
	}
	npmPkgs := []SoftwarePackage{
		{Name: "@anthropic-ai/sdk", Version: "0.24.0", Purl: "pkg:npm/%40anthropic-ai/sdk@0.24.0"},
		{Name: "ai", Version: "3.1.0", Purl: "pkg:npm/ai@3.1.0"},
		{Name: "express", Version: "4.19.0", Purl: "pkg:npm/express@4.19.0"},
	}

	deps := classifyAIDependencies(npmPkgs, pythonPkgs)

	// Should find: transformers, openai, langchain, chromadb, wandb (Python)
	// + @anthropic-ai/sdk, ai (npm) = 7 total
	// requests, flask, express are not AI packages
	assert.Len(t, deps, 7)

	byName := map[string]*aibom.AIDependency{}
	for _, d := range deps {
		byName[d.Name] = d
	}

	assert.Equal(t, "model-framework", byName["transformers"].Category)
	assert.Equal(t, "python", byName["transformers"].Language)
	assert.Equal(t, "4.40.0", byName["transformers"].Version)

	assert.Equal(t, "api-client", byName["openai"].Category)
	assert.Equal(t, "python", byName["openai"].Language)

	assert.Equal(t, "agent-framework", byName["langchain"].Category)
	assert.Equal(t, "vector-db", byName["chromadb"].Category)
	assert.Equal(t, "ml-tool", byName["wandb"].Category)

	assert.Equal(t, "api-client", byName["@anthropic-ai/sdk"].Category)
	assert.Equal(t, "javascript", byName["@anthropic-ai/sdk"].Language)

	assert.Equal(t, "api-client", byName["ai"].Category)
	assert.Equal(t, "javascript", byName["ai"].Language)

	// Non-AI packages should not appear
	_, hasRequests := byName["requests"]
	assert.False(t, hasRequests)
	_, hasExpress := byName["express"]
	assert.False(t, hasExpress)
}

func TestClassifyAIDependencies_Deduplication(t *testing.T) {
	pkgs := []SoftwarePackage{
		{Name: "openai", Version: "1.30.0"},
		{Name: "openai", Version: "1.30.0"},
	}

	deps := classifyAIDependencies(nil, pkgs)
	assert.Len(t, deps, 1)
}

func TestClassifyAIDependencies_Empty(t *testing.T) {
	deps := classifyAIDependencies(nil, nil)
	assert.Nil(t, deps)
}

func TestAIDependency_CycloneDX(t *testing.T) {
	pythonPkgs := []SoftwarePackage{
		{Name: "transformers", Version: "4.40.0", Purl: "pkg:pypi/transformers@4.40.0"},
		{Name: "langchain", Version: "0.2.0", Purl: "pkg:pypi/langchain@0.2.0"},
	}

	aiDeps := classifyAIDependencies(nil, pythonPkgs)

	bom := &aibom.AiBom{
		Generator: &aibom.Generator{
			Vendor:  "Mondoo, Inc.",
			Name:    "cnspec",
			Version: "11.0.0",
		},
		Asset: &aibom.Asset{
			Name: "python-project",
		},
		AIDependencies: aiDeps,
		Completeness:   &aibom.CompletenessScore{TotalScore: 0.3},
	}

	formatter := &aibom.CycloneDXFormatter{Format: 1}
	var buf bytes.Buffer
	err := formatter.Render(&buf, bom)
	require.NoError(t, err)

	var cdx map[string]any
	err = json.Unmarshal(buf.Bytes(), &cdx)
	require.NoError(t, err)

	components := cdx["components"].([]any)
	require.Len(t, components, 2)

	// transformers should be a library
	comp0 := components[0].(map[string]any)
	assert.Equal(t, "library", comp0["type"])
	assert.Equal(t, "transformers", comp0["name"])
	assert.Equal(t, "4.40.0", comp0["version"])
	assert.Equal(t, "pkg:pypi/transformers@4.40.0", comp0["purl"])

	// langchain should be a framework
	comp1 := components[1].(map[string]any)
	assert.Equal(t, "framework", comp1["type"])
	assert.Equal(t, "langchain", comp1["name"])

	// Verify properties
	props0 := comp0["properties"].([]any)
	propMap := map[string]string{}
	for _, p := range props0 {
		prop := p.(map[string]any)
		propMap[prop["name"].(string)] = prop["value"].(string)
	}
	assert.Equal(t, "model-framework", propMap["mondoo:ai:category"])
	assert.Equal(t, "python", propMap["mondoo:ai:language"])

	// Verify metadata includes ai_dependency_count
	metadata := cdx["metadata"].(map[string]any)
	metaProps := metadata["properties"].([]any)
	metaPropMap := map[string]string{}
	for _, p := range metaProps {
		prop := p.(map[string]any)
		metaPropMap[prop["name"].(string)] = prop["value"].(string)
	}
	assert.Equal(t, "2", metaPropMap["mondoo:aibom:ai_dependency_count"])
}

func TestPipelineTagToModalities(t *testing.T) {
	tests := []struct {
		tag           string
		expectInputs  []string
		expectOutputs []string
	}{
		{"automatic-speech-recognition", []string{"audio"}, []string{"text"}},
		{"text-generation", []string{"text"}, []string{"text"}},
		{"image-classification", []string{"image"}, []string{"text"}},
		{"text-to-image", []string{"text"}, []string{"image"}},
		{"text-to-speech", []string{"text"}, []string{"audio"}},
		{"feature-extraction", []string{"text"}, []string{"tensor"}},
		{"unknown-task", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			inputs, outputs := pipelineTagToModalities(tt.tag)
			assert.Equal(t, tt.expectInputs, inputs)
			assert.Equal(t, tt.expectOutputs, outputs)
		})
	}
}
