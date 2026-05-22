# ADR-0003: AI Bill of Materials (AIBOM) Generation

**Date:** 2026-05-22
**Status:** Proposed

## Context

Organizations deploying AI models across multiple clouds lack a unified inventory of what models they run, where they came from, what data they were trained on, and what licenses or ethical constraints apply.

The [CycloneDX](https://cyclonedx.org/) standard (ECMA-424) defines a full-stack Bill of Materials format with distinct BOM types: [SBOM](https://cyclonedx.org/capabilities/sbom/), [ML-BOM](https://cyclonedx.org/capabilities/mlbom/), [CBOM](https://cyclonedx.org/capabilities/cbom/), [HBOM](https://cyclonedx.org/capabilities/hbom/), and others. The [ML-BOM capability](https://cyclonedx.org/capabilities/mlbom/) (introduced in CycloneDX 1.5) provides machine-readable representation of AI/ML model transparency information including model parameters, datasets, performance metrics, fairness assessments, and ethical considerations — aligned with the [CycloneDX Machine Learning Model Card](https://cyclonedx.org/docs/1.6/json/#components_items_modelCard) schema.

cnspec already generates software SBOMs via `cnspec sbom` in CycloneDX, SPDX, and custom JSON formats. The CycloneDX Go library we depend on (`github.com/CycloneDX/cyclonedx-go` v0.11.0) already includes full ML-BOM support:

- [`ComponentTypeMachineLearningModel`](https://cyclonedx.org/docs/1.6/json/#components_items_type) and [`ComponentTypeData`](https://cyclonedx.org/docs/1.6/json/#components_items_type) component types
- [`MLModelCard`](https://cyclonedx.org/docs/1.6/json/#components_items_modelCard) with `MLModelParameters`, `MLQuantitativeAnalysis`, `MLModelCardConsiderations`
- [`MLModelCardEthicalConsideration`](https://cyclonedx.org/docs/1.6/json/#components_items_modelCard_considerations_ethicalConsiderations), `MLModelCardFairnessAssessment`
- [`ComponentData`](https://cyclonedx.org/docs/1.6/json/#components_items_data) for dataset representation

No library upgrade is needed.

## Decision

### 1. New `cnspec aibom` command

Add a new top-level `cnspec aibom` command, parallel to the existing `cnspec sbom`:

```bash
cnspec sbom local -o cyclonedx-json               # software BOM (unchanged)

# Cloud platforms
cnspec aibom aws -o cyclonedx-json                 # AI BOM from AWS (Bedrock + SageMaker)
cnspec aibom gcp -o cyclonedx-json                 # AI BOM from GCP (Vertex AI)
cnspec aibom azure -o cyclonedx-json               # AI BOM from Azure (OpenAI + ML)

# Model registries
cnspec aibom huggingface -o cyclonedx-json         # AI BOM from HuggingFace
cnspec aibom nvidia -o cyclonedx-json              # AI BOM from NVIDIA NGC

# ML platforms
cnspec aibom databricks -o cyclonedx-json          # AI BOM from Databricks
cnspec aibom mlflow -o cyclonedx-json              # AI BOM from MLflow

# AI inference APIs
cnspec aibom openai -o cyclonedx-json              # AI BOM from OpenAI
cnspec aibom anthropic -o cyclonedx-json           # AI BOM from Anthropic
cnspec aibom mistral -o cyclonedx-json             # AI BOM from Mistral AI
cnspec aibom cohere -o cyclonedx-json              # AI BOM from Cohere
cnspec aibom togetherai -o cyclonedx-json          # AI BOM from Together AI
cnspec aibom replicate -o cyclonedx-json           # AI BOM from Replicate
cnspec aibom groq -o cyclonedx-json                # AI BOM from Groq
cnspec aibom fireworks -o cyclonedx-json           # AI BOM from Fireworks AI

# Local runtimes
cnspec aibom ollama -o cyclonedx-json              # AI BOM from local Ollama
```

**Rationale:** CycloneDX defines [BOM as an umbrella standard](https://cyclonedx.org/capabilities/) with SBOM, ML-BOM, CBOM, HBOM, etc. as peer types. A dedicated `cnspec aibom` command keeps the naming clear and avoids overloading `cnspec sbom` with `--type` flags that change its fundamental behavior. The two commands share format rendering infrastructure (`cyclonedx.go`, `spdx.go`) but have separate query packs and generators.

### 2. Extend the data model with ML-specific messages

The existing `Sbom` message and `Package` type stay unchanged for software components. A new `AiBom` message is the AI-equivalent peer, reusing existing shared types (`Generator`, `Asset`, `Status`, `Platform`) but with its own top-level structure:

```protobuf
message AiBom {
  Generator generator = 1;
  string timestamp = 2;
  Status status = 3;
  Asset asset = 4;
  repeated ModelComponent models = 5;
  CompletenessScore completeness = 6;
  string error_message = 7;
}

message ModelComponent {
  string name = 1;
  string version = 2;
  string provider = 3;              // "huggingface", "aws-bedrock", "aws-sagemaker", "gcp-vertexai", "azure-openai"
  string model_id = 4;
  string description = 5;
  string author = 6;
  string license = 7;
  string task = 8;                   // "text-generation", "image-classification"
  string architecture_family = 9;
  string model_architecture = 10;
  string approach_type = 11;         // supervised, unsupervised, reinforcement-learning
  repeated string input_modalities = 12;
  repeated string output_modalities = 13;
  repeated string intended_uses = 14;
  repeated string limitations = 15;
  repeated string training_datasets = 16;
  repeated EthicalConsideration ethical_considerations = 17;
  map<string, string> performance_metrics = 18;
  map<string, string> labels = 19;
  string purl = 20;
  string source_url = 21;
  string created_at = 22;
  string updated_at = 23;
  map<string, string> provenance = 24;
}

message EthicalConsideration {
  string name = 1;
  string mitigation_strategy = 2;
}

message CompletenessScore {
  float total_score = 1;
  map<string, float> section_scores = 2;
  repeated string missing_fields = 3;
  repeated string recommendations = 4;
}
```

### 3. CycloneDX ML-BOM rendering

Each `ModelComponent` maps to a [`cyclonedx.Component`](https://cyclonedx.org/docs/1.6/json/#components_items) with:

- `Type`: [`machine-learning-model`](https://cyclonedx.org/docs/1.6/json/#components_items_type)
- [`ModelCard`](https://cyclonedx.org/docs/1.6/json/#components_items_modelCard):
  - [`ModelParameters`](https://cyclonedx.org/docs/1.6/json/#components_items_modelCard_modelParameters) — task, architectureFamily, modelArchitecture, approach, datasets, inputs/outputs
  - [`Considerations`](https://cyclonedx.org/docs/1.6/json/#components_items_modelCard_considerations) — useCases, technicalLimitations, ethicalConsiderations, fairnessAssessments
  - [`QuantitativeAnalysis`](https://cyclonedx.org/docs/1.6/json/#components_items_modelCard_quantitativeAnalysis) — performance metrics
- [`Licenses`](https://cyclonedx.org/docs/1.6/json/#components_items_licenses) from model license
- [`PackageURL`](https://cyclonedx.org/docs/1.6/json/#components_items_purl) with provider-specific PURL
- [`ExternalReferences`](https://cyclonedx.org/docs/1.6/json/#components_items_externalReferences) for source URL, model card URL

Training datasets render as nested components with `Type`: [`data`](https://cyclonedx.org/docs/1.6/json/#components_items_type) and populated [`ComponentData`](https://cyclonedx.org/docs/1.6/json/#components_items_data) fields.

### 4. Completeness scoring

Weighted scoring per model, averaged across the BOM:

| Section | Weight | Fields |
|---------|--------|--------|
| Identity | 0.25 | name, version, provider, model_id, author |
| License & Legal | 0.20 | license |
| Technical | 0.20 | task, architecture, approach, input/output modalities |
| Training & Data | 0.20 | training_datasets, provenance |
| Ethics & Risk | 0.15 | ethical_considerations, limitations, intended_uses |

Output as CycloneDX BOM-level [`properties`](https://cyclonedx.org/docs/1.6/json/#metadata_properties) and in cnspec-json as the `completeness` field.

### 5. Standalone AIBOM query pack

The AIBOM query pack lives in `content/mondoo-aibom.mql.yaml` as a separate, standalone query pack — not embedded in SBOM internals. This differs from the software SBOM approach (`internal/sbom/pack/sbom.mql.yaml`) because:

- **Multi-provider:** Queries are filtered by platform (`asset.platform`), so only relevant queries execute per scan target
- **Evolving:** New providers and fields will be added over time
- **Customizable:** Organizations may want to add custom queries or disable specific providers
- **Policy linting:** `cnspec policy lint` works on `content/` bundles

## cnspec Changes Required

| File | Change |
|------|--------|
| `apps/cnspec/cmd/aibom.go` | New: `cnspec aibom` command, loads `content/mondoo-aibom.mql.yaml` |
| `content/mondoo-aibom.mql.yaml` | New: standalone AIBOM query pack with per-provider queries |
| `internal/aibom/aibom.proto` | New: `AiBom`, `ModelComponent`, `EthicalConsideration`, `CompletenessScore` messages (imports shared types from `internal/sbom/`) |
| `mql/aibom/aibom.proto` | Same proto (shared definition for mql consumers) |
| `internal/aibom/cyclonedx.go` | New: ML-BOM component rendering per CycloneDX 1.6 ML-BOM spec |
| `internal/aibom/generator.go` | New: parse MQL report into `ModelComponent` protos per provider |
| `internal/aibom/scoring.go` | New: completeness scoring algorithm |

## MQL Provider Dependencies

The `cnspec aibom` command requires MQL resources to be available for each target provider. Some providers already have the necessary resources; others require new providers or new resources in existing providers. See the MQL requirements document for detailed resource definitions, API mappings, and phased rollout plan.

| Provider | MQL Status |
|----------|-----------|
| AWS Bedrock / SageMaker | Existing resources — no MQL changes needed |
| GCP Vertex AI | Existing resources — no MQL changes needed |
| Azure OpenAI + ML | New resources in existing Azure provider |
| HuggingFace | New provider required |
| Ollama | New provider required |
| NVIDIA NGC | New provider required |
| Databricks | New provider required |
| MLflow | New provider required |
| Together AI | New provider required |
| OpenAI | New provider required |
| Anthropic | New provider required |
| Mistral AI | New provider required |
| Cohere | New provider required |
| Replicate | New provider required |
| Groq | New provider required |
| Fireworks AI | New provider required |

## Consequences

**Advantages:**
- Dedicated `cnspec aibom` command — clear separation from `cnspec sbom`, aligned with [CycloneDX BOM taxonomy](https://cyclonedx.org/capabilities/)
- Continuous & policy-driven — unlike one-shot generators, cnspec can enforce completeness thresholds and security baselines across fleets of models
- Multi-cloud + local — single tool covers model registries (HuggingFace, NVIDIA NGC), cloud platforms (AWS, GCP, Azure), ML platforms (Databricks, MLflow), inference APIs (OpenAI, Anthropic, Mistral, Cohere, Together AI, Replicate, Groq, Fireworks AI), and local runtimes (Ollama)
- Standards-based — [CycloneDX ML-BOM](https://cyclonedx.org/capabilities/mlbom/) output interoperates with existing BOM tooling
- Shadow AI detection — Ollama provider complements ADR-0002's agent detection with local model inventory

**Risks:**
- **Metadata asymmetry:** HuggingFace completeness scores will be dramatically higher than Azure or Ollama. Scores reflect provider transparency, not security posture.
- **SageMaker model card parsing:** The `content()` field is a JSON string whose schema varies. Robust parsing with graceful fallbacks is needed.
- **HuggingFace model card parsing:** YAML frontmatter is structured but the markdown body is freeform. Extracting intended_uses/limitations from prose requires heuristic section detection.
- **13 new providers:** HuggingFace, Ollama, NVIDIA NGC, Databricks, MLflow, Together AI, OpenAI, Anthropic, Mistral, Cohere, Replicate, Groq, and Fireworks AI each require a full provider implementation in cnquery. This is the largest effort.
- **Azure SDK integration:** Adding new Azure resources requires new SDK dependencies.

## References

- [CycloneDX ML-BOM Capability](https://cyclonedx.org/capabilities/mlbom/)
- [CycloneDX 1.6 JSON Schema — modelCard](https://cyclonedx.org/docs/1.6/json/#components_items_modelCard)
- [CycloneDX 1.6 JSON Schema — component types](https://cyclonedx.org/docs/1.6/json/#components_items_type)
- [CycloneDX 1.6 JSON Schema — data](https://cyclonedx.org/docs/1.6/json/#components_items_data)
- [CycloneDX Specification (ECMA-424)](https://ecma-international.org/publications-and-standards/standards/ecma-424/)
- [cyclonedx-go library](https://github.com/CycloneDX/cyclonedx-go)
