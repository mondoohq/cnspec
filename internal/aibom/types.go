// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package aibom provides AI Bill of Materials generation.
//
// This file contains hand-written Go types that mirror the proto definitions
// in cnspec_aibom.proto. Once the proto is compiled via `make cnspec/generate`,
// this file should be replaced by the generated .pb.go code.
package aibom

type Status int32

const (
	Status_STATUS_UNSPECIFIED         Status = 0
	Status_STATUS_SUCCEEDED           Status = 1
	Status_STATUS_PARTIALLY_SUCCEEDED Status = 2
	Status_STATUS_FAILED              Status = 3
	Status_STATUS_STARTED             Status = 4
)

type AiBom struct {
	Generator      *Generator         `json:"generator,omitempty"`
	Timestamp      string             `json:"timestamp,omitempty"`
	Status         Status             `json:"status,omitempty"`
	Asset          *Asset             `json:"asset,omitempty"`
	Models         []*ModelComponent  `json:"models,omitempty"`
	Agents         []*AgentComponent  `json:"agents,omitempty"`
	Guardrails     []*Guardrail       `json:"guardrails,omitempty"`
	KnowledgeBases []*KnowledgeBase   `json:"knowledge_bases,omitempty"`
	ComputeAccess  []*ComputeAIAccess `json:"compute_access,omitempty"`
	AIDependencies []*AIDependency    `json:"ai_dependencies,omitempty"`
	Completeness   *CompletenessScore `json:"completeness,omitempty"`
	Errors         []string           `json:"errors,omitempty"`
}

type ComputeAIAccess struct {
	Name           string            `json:"name,omitempty"`
	Type           string            `json:"type,omitempty"`
	Provider       string            `json:"provider,omitempty"`
	Arn            string            `json:"arn,omitempty"`
	Runtime        string            `json:"runtime,omitempty"`
	ImageUri       string            `json:"image_uri,omitempty"`
	CodeHash       string            `json:"code_hash,omitempty"`
	ServiceAccount string            `json:"service_account,omitempty"`
	AIServices     []string          `json:"ai_services,omitempty"`
	AIActions      []string          `json:"ai_actions,omitempty"`
	EnvHints       map[string]string `json:"env_hints,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
}

type Generator struct {
	Vendor  string `json:"vendor,omitempty"`
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
	Url     string `json:"url,omitempty"`
}

type Asset struct {
	Name        string            `json:"name,omitempty"`
	PlatformIds []string          `json:"platform_ids,omitempty"`
	Platform    *Platform         `json:"platform,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

func (a *Asset) GetName() string {
	if a == nil {
		return ""
	}
	return a.Name
}

type Platform struct {
	Name    string   `json:"name,omitempty"`
	Version string   `json:"version,omitempty"`
	Arch    string   `json:"arch,omitempty"`
	Title   string   `json:"title,omitempty"`
	Family  []string `json:"family,omitempty"`
}

type ModelComponent struct {
	Name                  string                  `json:"name,omitempty"`
	Version               string                  `json:"version,omitempty"`
	Provider              string                  `json:"provider,omitempty"`
	ModelId               string                  `json:"model_id,omitempty"`
	Description           string                  `json:"description,omitempty"`
	Author                string                  `json:"author,omitempty"`
	License               string                  `json:"license,omitempty"`
	Task                  string                  `json:"task,omitempty"`
	ArchitectureFamily    string                  `json:"architecture_family,omitempty"`
	ModelArchitecture     string                  `json:"model_architecture,omitempty"`
	ApproachType          string                  `json:"approach_type,omitempty"`
	InputModalities       []string                `json:"input_modalities,omitempty"`
	OutputModalities      []string                `json:"output_modalities,omitempty"`
	IntendedUses          []string                `json:"intended_uses,omitempty"`
	Limitations           []string                `json:"limitations,omitempty"`
	TrainingDatasets      []string                `json:"training_datasets,omitempty"`
	EthicalConsiderations []*EthicalConsideration `json:"ethical_considerations,omitempty"`
	PerformanceMetrics    map[string]string       `json:"performance_metrics,omitempty"`
	Labels                map[string]string       `json:"labels,omitempty"`
	Purl                  string                  `json:"purl,omitempty"`
	SourceUrl             string                  `json:"source_url,omitempty"`
	CreatedAt             string                  `json:"created_at,omitempty"`
	UpdatedAt             string                  `json:"updated_at,omitempty"`
	Provenance            map[string]string       `json:"provenance,omitempty"`
	Format                string                  `json:"format,omitempty"`
	Quantization          string                  `json:"quantization,omitempty"`
	ParameterSize         string                  `json:"parameter_size,omitempty"`
	Tags                  []string                `json:"tags,omitempty"`
	Capabilities          []string                `json:"capabilities,omitempty"`
}

type AgentComponent struct {
	Name         string             `json:"name,omitempty"`
	Provider     string             `json:"provider,omitempty"`
	ConfigPath   string             `json:"config_path,omitempty"`
	Version      string             `json:"version,omitempty"`
	Model        string             `json:"model,omitempty"`
	McpServers   []*McpServer       `json:"mcp_servers,omitempty"`
	Plugins      []*AgentPlugin     `json:"plugins,omitempty"`
	Extensions   []*AgentExtension  `json:"extensions,omitempty"`
	Skills       []*AgentSkill      `json:"skills,omitempty"`
	Dependencies []*AgentDependency `json:"dependencies,omitempty"`
	Labels       map[string]string  `json:"labels,omitempty"`
}

type McpServer struct {
	Name     string        `json:"name,omitempty"`
	Type     string        `json:"type,omitempty"`
	Command  string        `json:"command,omitempty"`
	Args     []string      `json:"args,omitempty"`
	Url      string        `json:"url,omitempty"`
	HasEnv   bool          `json:"has_env,omitempty"`
	Purl     string        `json:"purl,omitempty"`
	Packages []*PackageRef `json:"packages,omitempty"`
}

type PackageRef struct {
	Name    string   `json:"name,omitempty"`
	Version string   `json:"version,omitempty"`
	Purl    string   `json:"purl,omitempty"`
	Cpes    []string `json:"cpes,omitempty"`
}

type AgentPlugin struct {
	Name         string `json:"name,omitempty"`
	Version      string `json:"version,omitempty"`
	Author       string `json:"author,omitempty"`
	Description  string `json:"description,omitempty"`
	InstallPath  string `json:"install_path,omitempty"`
	GitCommitSha string `json:"git_commit_sha,omitempty"`
	Enabled      bool   `json:"enabled,omitempty"`
	Purl         string `json:"purl,omitempty"`
}

type AgentExtension struct {
	Name        string `json:"name,omitempty"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled,omitempty"`
	Bundled     bool   `json:"bundled,omitempty"`
}

type AgentSkill struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source,omitempty"`
	Sha256      string `json:"sha256,omitempty"`
}

type Guardrail struct {
	Name     string            `json:"name,omitempty"`
	Provider string            `json:"provider,omitempty"`
	ID       string            `json:"id,omitempty"`
	Arn      string            `json:"arn,omitempty"`
	Status   string            `json:"status,omitempty"`
	Version  string            `json:"version,omitempty"`
	Policies []string          `json:"policies,omitempty"`
	Labels   map[string]string `json:"labels,omitempty"`
}

type KnowledgeBase struct {
	Name           string            `json:"name,omitempty"`
	Provider       string            `json:"provider,omitempty"`
	ID             string            `json:"id,omitempty"`
	Arn            string            `json:"arn,omitempty"`
	Status         string            `json:"status,omitempty"`
	Description    string            `json:"description,omitempty"`
	EmbeddingModel string            `json:"embedding_model,omitempty"`
	StorageType    string            `json:"storage_type,omitempty"`
	DataSources    []string          `json:"data_sources,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
}

type AgentDependency struct {
	Type     string            `json:"type,omitempty"`
	Name     string            `json:"name,omitempty"`
	Arn      string            `json:"arn,omitempty"`
	Runtime  string            `json:"runtime,omitempty"`
	ImageUri string            `json:"image_uri,omitempty"`
	CodeHash string            `json:"code_hash,omitempty"`
	Layers   []string          `json:"layers,omitempty"`
	Labels   map[string]string `json:"labels,omitempty"`
}

type AIDependency struct {
	Name     string `json:"name,omitempty"`
	Version  string `json:"version,omitempty"`
	Category string `json:"category,omitempty"`
	Purl     string `json:"purl,omitempty"`
	Language string `json:"language,omitempty"`
}

type EthicalConsideration struct {
	Name               string `json:"name,omitempty"`
	MitigationStrategy string `json:"mitigation_strategy,omitempty"`
}

type CompletenessScore struct {
	TotalScore      float32            `json:"total_score,omitempty"`
	SectionScores   map[string]float32 `json:"section_scores,omitempty"`
	MissingFields   []string           `json:"missing_fields,omitempty"`
	Recommendations []string           `json:"recommendations,omitempty"`
}

func (c *CompletenessScore) GetTotalScore() float32 {
	if c == nil {
		return 0
	}
	return c.TotalScore
}
