// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package aibom

import (
	"fmt"
	"io"
	"strings"
	"time"

	cyclonedx "github.com/CycloneDX/cyclonedx-go"
	"github.com/google/uuid"
)

type CycloneDXFormatter struct {
	Format cyclonedx.BOMFileFormat
}

func (f *CycloneDXFormatter) Render(w io.Writer, bom *AiBom) error {
	cdxBom, err := f.convert(bom)
	if err != nil {
		return err
	}
	encoder := cyclonedx.NewBOMEncoder(w, f.Format)
	encoder.SetPretty(true)
	return encoder.Encode(cdxBom)
}

func (f *CycloneDXFormatter) convert(bom *AiBom) (*cyclonedx.BOM, error) {
	cdx := cyclonedx.NewBOM()
	cdx.SerialNumber = uuid.New().URN()
	cdx.Metadata = &cyclonedx.Metadata{
		Timestamp: time.Now().Format(time.RFC3339),
		Tools: &cyclonedx.ToolsChoice{
			Components: &[]cyclonedx.Component{
				{
					Type:    cyclonedx.ComponentTypeApplication,
					Author:  bom.Generator.Vendor,
					Name:    bom.Generator.Name,
					Version: bom.Generator.Version,
				},
			},
		},
		Properties: &[]cyclonedx.Property{
			{Name: "mondoo:aibom:completeness:total", Value: fmt.Sprintf("%.2f", bom.Completeness.GetTotalScore())},
			{Name: "mondoo:aibom:model_count", Value: fmt.Sprintf("%d", len(bom.Models))},
			{Name: "mondoo:aibom:agent_count", Value: fmt.Sprintf("%d", len(bom.Agents))},
			{Name: "mondoo:aibom:guardrail_count", Value: fmt.Sprintf("%d", len(bom.Guardrails))},
			{Name: "mondoo:aibom:knowledge_base_count", Value: fmt.Sprintf("%d", len(bom.KnowledgeBases))},
			{Name: "mondoo:aibom:ai_dependency_count", Value: fmt.Sprintf("%d", len(bom.AIDependencies))},
		},
	}

	if bom.Asset != nil {
		cdx.Metadata.Component = &cyclonedx.Component{
			BOMRef: uuid.New().String(),
			Type:   cyclonedx.ComponentTypePlatform,
			Name:   bom.Asset.Name,
		}
	}

	components := make([]cyclonedx.Component, 0, len(bom.Models)+len(bom.Agents)+len(bom.Guardrails)+len(bom.KnowledgeBases)+len(bom.ComputeAccess)+len(bom.AIDependencies))
	for _, model := range bom.Models {
		components = append(components, modelToComponent(model))
	}
	for _, agent := range bom.Agents {
		components = append(components, agentToComponent(agent))
	}
	for _, g := range bom.Guardrails {
		components = append(components, guardrailToComponent(g))
	}
	for _, kb := range bom.KnowledgeBases {
		components = append(components, knowledgeBaseToComponent(kb))
	}
	for _, ca := range bom.ComputeAccess {
		components = append(components, computeAccessToComponent(ca))
	}
	for _, dep := range bom.AIDependencies {
		components = append(components, aiDependencyToComponent(dep))
	}
	cdx.Components = &components

	return cdx, nil
}

func modelToComponent(m *ModelComponent) cyclonedx.Component {
	c := cyclonedx.Component{
		Type:    cyclonedx.ComponentTypeMachineLearningModel,
		Name:    m.Name,
		Version: m.Version,
	}

	if m.Purl != "" {
		c.BOMRef = m.Purl
		c.PackageURL = m.Purl
	} else {
		c.BOMRef = uuid.New().String()
	}

	if m.Description != "" {
		c.Description = m.Description
	}

	// Set group, supplier, manufacturer, and authors from Author field
	if m.Author != "" {
		c.Group = m.Author
		c.Supplier = &cyclonedx.OrganizationalEntity{Name: m.Author}
		c.Manufacturer = &cyclonedx.OrganizationalEntity{Name: m.Author}
		authors := []cyclonedx.OrganizationalContact{{Name: m.Author}}
		c.Authors = &authors

		if m.SourceUrl != "" {
			c.Manufacturer.URL = &[]string{m.SourceUrl}
		}
	}

	// Licenses — use SPDX ID for well-known licenses
	if m.License != "" {
		license := cyclonedx.LicenseChoice{}
		if isSPDXLicense(m.License) {
			license.License = &cyclonedx.License{ID: spdxNormalize(m.License)}
		} else {
			license.License = &cyclonedx.License{Name: m.License}
		}
		c.Licenses = &cyclonedx.Licenses{license}
	}

	// External references
	refs := []cyclonedx.ExternalReference{}
	if m.SourceUrl != "" {
		refs = append(refs, cyclonedx.ExternalReference{
			Type:    cyclonedx.ERTypeWebsite,
			URL:     m.SourceUrl,
			Comment: "Model repository",
		})
	}
	// Distribution and VCS links only for HuggingFace-style repo URLs
	if m.SourceUrl != "" && strings.Contains(m.SourceUrl, "huggingface.co/") {
		refs = append(refs, cyclonedx.ExternalReference{
			Type:    cyclonedx.ERTypeDistribution,
			URL:     m.SourceUrl + "/tree/main",
			Comment: "Model files",
		})
		if sha, ok := m.Provenance["sha"]; ok {
			refs = append(refs, cyclonedx.ExternalReference{
				Type:    cyclonedx.ERTypeVCS,
				URL:     m.SourceUrl + "/commit/" + sha,
				Comment: "Specific commit",
			})
		}
	} else if m.SourceUrl != "" && !strings.HasPrefix(m.SourceUrl, "file://") {
		refs = append(refs, cyclonedx.ExternalReference{
			Type:    cyclonedx.ERTypeDistribution,
			URL:     m.SourceUrl,
			Comment: "Model endpoint",
		})
	}
	// ArXiv paper reference
	if arxiv, ok := m.Provenance["arxiv"]; ok {
		refs = append(refs, cyclonedx.ExternalReference{
			Type:    cyclonedx.ERTypeDocumentation,
			URL:     "https://arxiv.org/abs/" + arxiv,
			Comment: "ArXiv Paper",
		})
	}
	// Training dataset references
	for _, ds := range m.TrainingDatasets {
		refs = append(refs, cyclonedx.ExternalReference{
			Type:    cyclonedx.ERTypeDistribution,
			URL:     "https://huggingface.co/datasets/" + ds,
			Comment: "Training dataset: " + ds,
		})
	}
	if len(refs) > 0 {
		c.ExternalReferences = &refs
	}

	// ModelCard
	mc := &cyclonedx.MLModelCard{}
	hasModelCard := false

	// ModelParameters
	params := &cyclonedx.MLModelParameters{}
	hasParams := false

	if m.Task != "" {
		params.Task = m.Task
		hasParams = true
	}
	if m.ArchitectureFamily != "" {
		params.ArchitectureFamily = m.ArchitectureFamily
		hasParams = true
	}
	if m.ModelArchitecture != "" {
		params.ModelArchitecture = m.ModelArchitecture
		hasParams = true
	}
	if m.ApproachType != "" {
		params.Approach = &cyclonedx.MLModelParametersApproach{
			Type: cyclonedx.MLModelParametersApproachType(m.ApproachType),
		}
		hasParams = true
	}

	// Input/output modalities
	if len(m.InputModalities) > 0 {
		inputs := &[]cyclonedx.MLInputOutputParameters{}
		for _, mod := range m.InputModalities {
			*inputs = append(*inputs, cyclonedx.MLInputOutputParameters{
				Format: mod,
			})
		}
		params.Inputs = inputs
		hasParams = true
	}
	if len(m.OutputModalities) > 0 {
		outputs := &[]cyclonedx.MLInputOutputParameters{}
		for _, mod := range m.OutputModalities {
			*outputs = append(*outputs, cyclonedx.MLInputOutputParameters{
				Format: mod,
			})
		}
		params.Outputs = outputs
		hasParams = true
	}

	// Training datasets
	if len(m.TrainingDatasets) > 0 {
		datasets := &[]cyclonedx.MLDatasetChoice{}
		for _, ds := range m.TrainingDatasets {
			*datasets = append(*datasets, cyclonedx.MLDatasetChoice{
				ComponentData: &cyclonedx.ComponentData{
					Type: cyclonedx.ComponentDataTypeDataset,
					Name: ds,
					Contents: &cyclonedx.ComponentDataContents{
						URL: "https://huggingface.co/datasets/" + ds,
					},
				},
			})
		}
		params.Datasets = datasets
		hasParams = true
	}

	if hasParams {
		mc.ModelParameters = params
		hasModelCard = true
	}

	// Considerations
	considerations := &cyclonedx.MLModelCardConsiderations{}
	hasConsiderations := false

	if len(m.IntendedUses) > 0 {
		uses := make([]string, len(m.IntendedUses))
		copy(uses, m.IntendedUses)
		considerations.UseCases = &uses
		hasConsiderations = true
	}
	if len(m.Limitations) > 0 {
		lims := make([]string, len(m.Limitations))
		copy(lims, m.Limitations)
		considerations.TechnicalLimitations = &lims
		hasConsiderations = true
	}
	if len(m.EthicalConsiderations) > 0 {
		ethics := &[]cyclonedx.MLModelCardEthicalConsideration{}
		for _, ec := range m.EthicalConsiderations {
			*ethics = append(*ethics, cyclonedx.MLModelCardEthicalConsideration{
				Name:               ec.Name,
				MitigationStrategy: ec.MitigationStrategy,
			})
		}
		considerations.EthicalConsiderations = ethics
		hasConsiderations = true
	}

	if hasConsiderations {
		mc.Considerations = considerations
		hasModelCard = true
	}

	// Quantitative analysis (performance metrics)
	if len(m.PerformanceMetrics) > 0 {
		metrics := &[]cyclonedx.MLPerformanceMetric{}
		for name, value := range m.PerformanceMetrics {
			*metrics = append(*metrics, cyclonedx.MLPerformanceMetric{
				Type:  name,
				Value: value,
			})
		}
		mc.QuantitativeAnalysis = &cyclonedx.MLQuantitativeAnalysis{
			PerformanceMetrics: metrics,
		}
		hasModelCard = true
	}

	if hasModelCard {
		c.ModelCard = mc
	}

	// Properties for fields not covered by CycloneDX ML-BOM schema
	props := []cyclonedx.Property{
		{Name: "mondoo:model:provider", Value: m.Provider},
	}
	if m.Format != "" {
		props = append(props, cyclonedx.Property{Name: "mondoo:model:format", Value: m.Format})
	}
	if m.Quantization != "" {
		props = append(props, cyclonedx.Property{Name: "mondoo:model:quantization", Value: m.Quantization})
	}
	if m.ParameterSize != "" {
		props = append(props, cyclonedx.Property{Name: "mondoo:model:parameterSize", Value: m.ParameterSize})
	}
	if len(m.Capabilities) > 0 {
		for _, cap := range m.Capabilities {
			props = append(props, cyclonedx.Property{Name: "mondoo:model:capability", Value: cap})
		}
	}
	for k, v := range m.Labels {
		props = append(props, cyclonedx.Property{Name: "genai:aibom:modelcard:" + k, Value: v})
	}
	for k, v := range m.Provenance {
		props = append(props, cyclonedx.Property{Name: "mondoo:model:provenance:" + k, Value: v})
	}
	c.Properties = &props

	return c
}

var spdxLicenses = map[string]string{
	"apache-2.0":   "Apache-2.0",
	"mit":          "MIT",
	"gpl-3.0":      "GPL-3.0-only",
	"gpl-2.0":      "GPL-2.0-only",
	"bsd-2-clause": "BSD-2-Clause",
	"bsd-3-clause": "BSD-3-Clause",
	"lgpl-3.0":     "LGPL-3.0-only",
	"lgpl-2.1":     "LGPL-2.1-only",
	"mpl-2.0":      "MPL-2.0",
	"isc":          "ISC",
	"unlicense":    "Unlicense",
	"cc-by-4.0":    "CC-BY-4.0",
	"cc-by-sa-4.0": "CC-BY-SA-4.0",
	"cc-by-nc-4.0": "CC-BY-NC-4.0",
	"cc0-1.0":      "CC0-1.0",
}

func isSPDXLicense(license string) bool {
	_, ok := spdxLicenses[strings.ToLower(license)]
	return ok
}

func spdxNormalize(license string) string {
	if id, ok := spdxLicenses[strings.ToLower(license)]; ok {
		return id
	}
	return license
}

func aiDependencyToComponent(dep *AIDependency) cyclonedx.Component {
	compType := cyclonedx.ComponentTypeLibrary
	if dep.Category == "agent-framework" {
		compType = cyclonedx.ComponentTypeFramework
	}

	c := cyclonedx.Component{
		Type:    compType,
		Name:    dep.Name,
		Version: dep.Version,
	}

	if dep.Purl != "" {
		c.BOMRef = dep.Purl
		c.PackageURL = dep.Purl
	} else {
		c.BOMRef = uuid.New().String()
	}

	props := []cyclonedx.Property{
		{Name: "mondoo:ai:category", Value: dep.Category},
	}
	if dep.Language != "" {
		props = append(props, cyclonedx.Property{Name: "mondoo:ai:language", Value: dep.Language})
	}
	c.Properties = &props

	return c
}

func computeAccessToComponent(ca *ComputeAIAccess) cyclonedx.Component {
	c := cyclonedx.Component{
		BOMRef: uuid.New().String(),
		Type:   cyclonedx.ComponentTypeApplication,
		Name:   ca.Name,
	}
	props := []cyclonedx.Property{
		{Name: "mondoo:compute:type", Value: ca.Type},
		{Name: "mondoo:compute:provider", Value: ca.Provider},
	}
	if ca.Runtime != "" {
		props = append(props, cyclonedx.Property{Name: "mondoo:compute:runtime", Value: ca.Runtime})
	}
	if ca.Arn != "" {
		props = append(props, cyclonedx.Property{Name: "mondoo:compute:arn", Value: ca.Arn})
	}
	if ca.ImageUri != "" {
		props = append(props, cyclonedx.Property{Name: "mondoo:compute:imageUri", Value: ca.ImageUri})
	}
	if ca.CodeHash != "" {
		props = append(props, cyclonedx.Property{Name: "mondoo:compute:codeSha256", Value: ca.CodeHash})
	}
	if ca.ServiceAccount != "" {
		props = append(props, cyclonedx.Property{Name: "mondoo:compute:serviceAccount", Value: ca.ServiceAccount})
	}
	for _, svc := range ca.AIServices {
		props = append(props, cyclonedx.Property{Name: "mondoo:compute:aiService", Value: svc})
	}
	for _, action := range ca.AIActions {
		props = append(props, cyclonedx.Property{Name: "mondoo:compute:aiAction", Value: action})
	}
	for k, v := range ca.EnvHints {
		props = append(props, cyclonedx.Property{Name: "mondoo:compute:envHint:" + k, Value: v})
	}
	c.Properties = &props
	return c
}

func agentToComponent(a *AgentComponent) cyclonedx.Component {
	agentRef := uuid.New().String()
	c := cyclonedx.Component{
		BOMRef:  agentRef,
		Type:    cyclonedx.ComponentTypeApplication,
		Name:    a.Name,
		Version: a.Version,
	}

	props := []cyclonedx.Property{
		{Name: "mondoo:agent:provider", Value: a.Provider},
		{Name: "mondoo:agent:type", Value: "coding-agent"},
	}
	if a.ConfigPath != "" {
		props = append(props, cyclonedx.Property{Name: "mondoo:agent:configPath", Value: a.ConfigPath})
	}
	if a.Model != "" {
		props = append(props, cyclonedx.Property{Name: "mondoo:agent:model", Value: a.Model})
	}
	for _, s := range a.Skills {
		p := cyclonedx.Property{Name: "mondoo:agent:skill", Value: s.Name}
		props = append(props, p)
		if s.Sha256 != "" {
			props = append(props, cyclonedx.Property{Name: "mondoo:agent:skill:sha256:" + s.Name, Value: s.Sha256})
		}
	}
	for k, v := range a.Labels {
		props = append(props, cyclonedx.Property{Name: "mondoo:agent:label:" + k, Value: v})
	}

	// Nested components: plugins + MCP servers form the agent's SBOM
	var nested []cyclonedx.Component

	for _, p := range a.Plugins {
		pc := cyclonedx.Component{
			BOMRef:  uuid.New().String(),
			Type:    cyclonedx.ComponentTypeLibrary,
			Name:    p.Name,
			Version: p.Version,
			Author:  p.Author,
		}
		if p.Purl != "" {
			pc.PackageURL = p.Purl
		}
		if p.Description != "" {
			pc.Description = p.Description
		}
		pprops := []cyclonedx.Property{
			{Name: "mondoo:plugin:agent", Value: a.Name},
		}
		if p.GitCommitSha != "" {
			pprops = append(pprops, cyclonedx.Property{Name: "mondoo:plugin:gitCommitSha", Value: p.GitCommitSha})
		}
		if p.InstallPath != "" {
			pprops = append(pprops, cyclonedx.Property{Name: "mondoo:plugin:installPath", Value: p.InstallPath})
		}
		pc.Properties = &pprops
		nested = append(nested, pc)
	}

	for _, m := range a.McpServers {
		mc := cyclonedx.Component{
			BOMRef: uuid.New().String(),
			Type:   cyclonedx.ComponentTypeApplication,
			Name:   "mcp:" + m.Name,
		}
		if m.Purl != "" {
			mc.PackageURL = m.Purl
		}
		mprops := []cyclonedx.Property{
			{Name: "mondoo:mcp:agent", Value: a.Name},
		}
		if m.Command != "" {
			mprops = append(mprops, cyclonedx.Property{Name: "mondoo:mcp:command", Value: m.Command})
		}
		if m.Url != "" {
			mprops = append(mprops, cyclonedx.Property{Name: "mondoo:mcp:url", Value: m.Url})
		}
		if m.HasEnv {
			mprops = append(mprops, cyclonedx.Property{Name: "mondoo:mcp:hasEnv", Value: "true"})
		}
		mc.Properties = &mprops
		nested = append(nested, mc)
	}

	for _, e := range a.Extensions {
		ec := cyclonedx.Component{
			BOMRef: uuid.New().String(),
			Type:   cyclonedx.ComponentTypeLibrary,
			Name:   e.Name,
		}
		if e.Description != "" {
			ec.Description = e.Description
		}
		eprops := []cyclonedx.Property{
			{Name: "mondoo:extension:agent", Value: a.Name},
			{Name: "mondoo:extension:type", Value: e.Type},
		}
		ec.Properties = &eprops
		nested = append(nested, ec)
	}

	for _, d := range a.Dependencies {
		dc := cyclonedx.Component{
			BOMRef: uuid.New().String(),
			Name:   d.Name,
		}
		switch d.Type {
		case "action-group", "cloud-function":
			dc.Type = cyclonedx.ComponentTypeApplication
		case "knowledge-base":
			dc.Type = cyclonedx.ComponentTypeData
		default:
			dc.Type = cyclonedx.ComponentTypeApplication
		}
		dprops := []cyclonedx.Property{
			{Name: "mondoo:dependency:agent", Value: a.Name},
			{Name: "mondoo:dependency:type", Value: d.Type},
		}
		if d.Arn != "" {
			dprops = append(dprops, cyclonedx.Property{Name: "mondoo:dependency:arn", Value: d.Arn})
		}
		if d.Runtime != "" {
			dprops = append(dprops, cyclonedx.Property{Name: "mondoo:dependency:runtime", Value: d.Runtime})
		}
		if d.ImageUri != "" {
			dprops = append(dprops, cyclonedx.Property{Name: "mondoo:dependency:imageUri", Value: d.ImageUri})
		}
		if d.CodeHash != "" {
			dprops = append(dprops, cyclonedx.Property{Name: "mondoo:dependency:codeSha256", Value: d.CodeHash})
		}
		for _, layer := range d.Layers {
			dprops = append(dprops, cyclonedx.Property{Name: "mondoo:dependency:layer", Value: layer})
		}
		for k, v := range d.Labels {
			dprops = append(dprops, cyclonedx.Property{Name: "mondoo:dependency:" + k, Value: v})
		}
		dc.Properties = &dprops
		nested = append(nested, dc)
	}

	if len(nested) > 0 {
		c.Components = &nested
	}

	c.Properties = &props
	return c
}

func guardrailToComponent(g *Guardrail) cyclonedx.Component {
	c := cyclonedx.Component{
		BOMRef:  uuid.New().String(),
		Type:    cyclonedx.ComponentTypeApplication,
		Name:    g.Name,
		Version: g.Version,
	}
	props := []cyclonedx.Property{
		{Name: "mondoo:guardrail:provider", Value: g.Provider},
		{Name: "mondoo:guardrail:status", Value: g.Status},
	}
	if g.Arn != "" {
		props = append(props, cyclonedx.Property{Name: "mondoo:guardrail:arn", Value: g.Arn})
	}
	for _, p := range g.Policies {
		props = append(props, cyclonedx.Property{Name: "mondoo:guardrail:policy", Value: p})
	}
	c.Properties = &props
	return c
}

func knowledgeBaseToComponent(kb *KnowledgeBase) cyclonedx.Component {
	c := cyclonedx.Component{
		BOMRef:      uuid.New().String(),
		Type:        cyclonedx.ComponentTypeData,
		Name:        kb.Name,
		Description: kb.Description,
	}
	props := []cyclonedx.Property{
		{Name: "mondoo:knowledgeBase:provider", Value: kb.Provider},
		{Name: "mondoo:knowledgeBase:status", Value: kb.Status},
	}
	if kb.Arn != "" {
		props = append(props, cyclonedx.Property{Name: "mondoo:knowledgeBase:arn", Value: kb.Arn})
	}
	if kb.EmbeddingModel != "" {
		props = append(props, cyclonedx.Property{Name: "mondoo:knowledgeBase:embeddingModel", Value: kb.EmbeddingModel})
	}
	if kb.StorageType != "" {
		props = append(props, cyclonedx.Property{Name: "mondoo:knowledgeBase:storageType", Value: kb.StorageType})
	}
	for _, ds := range kb.DataSources {
		props = append(props, cyclonedx.Property{Name: "mondoo:knowledgeBase:dataSource", Value: ds})
	}
	c.Properties = &props
	return c
}
