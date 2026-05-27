# ADR-0003: AI Bill of Materials (AIBOM) Generation

**Date:** 2026-05-22
**Status:** Accepted

## Context

Organizations deploying AI models across multiple clouds lack a unified inventory of what models they run, where they came from, what data they were trained on, and what licenses or ethical constraints apply. Beyond models, the proliferation of AI coding agents, MCP servers, guardrails, knowledge bases, and AI-capable compute creates a shadow AI problem — teams adopt AI tools without centralized visibility.

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

# Local systems
cnspec aibom local -o cyclonedx-json               # AI BOM from local machine (agents, cached models)
cnspec aibom local -o table                         # human-readable summary

# Cloud platforms
cnspec aibom aws -o cyclonedx-json                 # AI BOM from AWS (Bedrock + SageMaker + Lambda)
cnspec aibom gcp -o cyclonedx-json                 # AI BOM from GCP (Vertex AI + Model Armor + Cloud Functions/Run)
cnspec aibom azure -o cyclonedx-json               # AI BOM from Azure (OpenAI + Cognitive Services + Functions)

# Model registries and inference APIs
cnspec aibom ollama -o cyclonedx-json              # AI BOM from local Ollama
cnspec aibom huggingface -o cyclonedx-json         # AI BOM from HuggingFace
cnspec aibom openai -o cyclonedx-json              # AI BOM from OpenAI API (models, vector stores, fine-tuning)
cnspec aibom claude -o cyclonedx-json              # AI BOM from Anthropic Claude API (models, agents, skills)
cnspec aibom vllm -o cyclonedx-json                # AI BOM from vLLM server

# Filesystem / remote
cnspec aibom filesystem --path /path -o table      # scan filesystem for cached models
cnspec aibom ssh user@host -o cyclonedx-json       # remote host agent & model discovery
cnspec aibom docker image:tag -o cyclonedx-json    # container AI BOM
```

**Rationale:** CycloneDX defines [BOM as an umbrella standard](https://cyclonedx.org/capabilities/) with SBOM, ML-BOM, CBOM, HBOM, etc. as peer types. A dedicated `cnspec aibom` command keeps the naming clear and avoids overloading `cnspec sbom` with `--type` flags that change its fundamental behavior. The two commands share format rendering infrastructure (`cyclonedx.go`) but have separate query packs and generators.

### 2. Data model

The AIBOM data model extends beyond models to capture the full AI supply chain. Hand-written Go types in `internal/aibom/types.go` mirror proto definitions in `internal/aibom/cnspec_aibom.proto`:

```protobuf
message AiBom {
  Generator generator = 1;
  string timestamp = 2;
  Status status = 3;
  Asset asset = 4;
  repeated ModelComponent models = 5;
  repeated AgentComponent agents = 6;
  repeated Guardrail guardrails = 7;
  repeated KnowledgeBase knowledge_bases = 8;
  repeated ComputeAIAccess compute_access = 9;
  repeated AIDependency ai_dependencies = 10;
  CompletenessScore completeness = 11;
  repeated string errors = 12;
}
```

**ModelComponent** — AI/ML models from cloud providers, registries, local caches, and inference APIs:

```protobuf
message ModelComponent {
  string name = 1;
  string version = 2;
  string provider = 3;              // "ollama", "huggingface", "aws-bedrock", "aws-sagemaker",
                                     // "gcp-vertexai", "azure-openai", "openai", "anthropic", "vllm"
  string model_id = 4;
  string description = 5;
  string author = 6;
  string license = 7;
  string task = 8;
  string architecture_family = 9;
  string model_architecture = 10;
  string approach_type = 11;
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
  string format = 25;               // "gguf", "safetensors", "onnx", etc.
  string quantization = 26;
  string parameter_size = 27;
  repeated string tags = 28;
  repeated string capabilities = 29;
}
```

**AgentComponent** — AI coding agents (25 types) with MCP servers, plugins, extensions, and skills:

```protobuf
message AgentComponent {
  string name = 1;
  string provider = 2;              // "local", "anthropic", "aws-bedrock", "gcp-cloud-functions"
  string config_path = 3;
  string version = 4;
  string model = 5;
  repeated McpServer mcp_servers = 6;
  repeated AgentPlugin plugins = 7;
  repeated AgentExtension extensions = 8;
  repeated AgentSkill skills = 9;
  repeated AgentDependency dependencies = 10;
  map<string, string> labels = 11;
}
```

**Guardrail** — Safety guardrails from AWS Bedrock, GCP Model Armor, and Azure RAI policies.

**KnowledgeBase** — Vector stores and knowledge bases from AWS Bedrock, OpenAI, and Claude memory stores.

**ComputeAIAccess** — Compute services (Lambda, Cloud Functions, Cloud Run, Azure Functions) with IAM roles or environment variables that indicate AI service access. Environment variable values are redacted to `"(configured)"` to prevent API key leakage.

**AIDependency** — AI/ML libraries detected in npm and Python packages, classified by category (model-framework, api-client, agent-framework, vector-db, ml-tool).

### 3. CycloneDX ML-BOM rendering

Each `ModelComponent` maps to a [`cyclonedx.Component`](https://cyclonedx.org/docs/1.6/json/#components_items) with:

- `Type`: [`machine-learning-model`](https://cyclonedx.org/docs/1.6/json/#components_items_type)
- [`ModelCard`](https://cyclonedx.org/docs/1.6/json/#components_items_modelCard):
  - [`ModelParameters`](https://cyclonedx.org/docs/1.6/json/#components_items_modelCard_modelParameters) — task, architectureFamily, modelArchitecture, approach, datasets, inputs/outputs
  - [`Considerations`](https://cyclonedx.org/docs/1.6/json/#components_items_modelCard_considerations) — useCases, technicalLimitations, ethicalConsiderations, fairnessAssessments
  - [`QuantitativeAnalysis`](https://cyclonedx.org/docs/1.6/json/#components_items_modelCard_quantitativeAnalysis) — performance metrics
- [`Licenses`](https://cyclonedx.org/docs/1.6/json/#components_items_licenses) from model license (SPDX-normalized)
- [`PackageURL`](https://cyclonedx.org/docs/1.6/json/#components_items_purl) with provider-specific PURL
- [`ExternalReferences`](https://cyclonedx.org/docs/1.6/json/#components_items_externalReferences) for source URL, model card URL

Training datasets render as nested components with `Type`: [`data`](https://cyclonedx.org/docs/1.6/json/#components_items_type) and populated [`ComponentData`](https://cyclonedx.org/docs/1.6/json/#components_items_data) fields. Local file paths (`file://`) are excluded from distribution references.

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

### 5. Embedded AIBOM query pack

The AIBOM query pack is embedded in `internal/aibom/pack/aibom.mql.yaml` and loaded at runtime via `internal/aibom/pack/pack.go`. Queries are platform-filtered so only relevant queries execute per scan target:

- `asset.family.contains("unix") || asset.family.contains("windows")` — local agents, cached models, npm/python packages
- `asset.platform == "aws"` — Bedrock, SageMaker, Lambda
- `asset.platform == "gcp"` — Vertex AI, Model Armor, Cloud Functions, Cloud Run, IAM
- `asset.platform == "azure"` — Cognitive Services, Function Apps, RAI policies
- `asset.platform == "ollama"` — Ollama models with license detection
- `asset.family.contains("huggingface")` — HuggingFace models with card data
- `asset.platform == "openai"` — OpenAI models, vector stores, fine-tuning jobs
- `asset.platform == "claude"` — Claude models, agents, skills, memory stores, vaults
- `asset.platform == "vllm"` — vLLM server and models

### 6. Local AI agent and model discovery

**Coding agents** (25 types): Claude Code, OpenAI Codex, Cursor, GitHub Copilot, Windsurf, Gemini, Goose, Zed, Roo, Cline, Kiro, Trae, Junie, Augment, Kilocode, Continue, Mistral Vibe, Antigravity, IBM Bob, OpenClaw, Snowflake Cortex, Warp, OpenHands, OpenCode, Pi, Qwen Code. Each agent's query pack extracts config path, skills, MCP servers, and (where available) plugins and extensions.

**Local model cache detection** (`ai.models` resource): 8 filesystem detectors scan for cached AI models from Ollama, HuggingFace, LM Studio, GPT4All, PyTorch Hub, Keras, TFHub, and Jan. Extracts name, source, vendor, family, format, version, quantization, parameter size, architecture, license, and tags.

**AI dependency classification**: npm and Python packages are classified against a registry of ~80 known AI/ML libraries into categories: model-framework, api-client, agent-framework, vector-db, and ml-tool.

**MCP server package correlation**: Agent MCP server commands (npx, uvx, pipx, etc.) are correlated with discovered npm/Python packages to enrich servers with version, PURL, and CPE data.

### 7. Compute AI access detection

Lambda functions, GCP Cloud Functions, GCP Cloud Run services, and Azure Function Apps are analyzed for AI service access through:

- **IAM policy analysis**: AWS Lambda roles scanned for `bedrock:*`, `sagemaker:*`, `comprehend:*`, etc. action prefixes; GCP service accounts checked for `roles/aiplatform.*` and `roles/ml.*` roles
- **Environment variable hints**: Keys containing `BEDROCK`, `SAGEMAKER`, `OPENAI`, `ANTHROPIC`, `VERTEX`, `AI_MODEL`, `MODEL_ID`, `LLM`, `GEMINI`, `CLAUDE`, `HUGGINGFACE`, `INFERENCE`, `EMBEDDING` — values redacted to `"(configured)"`
- **Bedrock agent enrichment**: Agent action-group Lambda ARNs are resolved to enrich with runtime, code hash, image URI, and layers

## cnspec Changes

| File | Change |
|------|--------|
| `apps/cnspec/cmd/aibom.go` | `cnspec aibom` command with format flags (table, json, cyclonedx-json) |
| `apps/cnspec/cmd/root.go` | Register `aibomCmd` with supported connectors |
| `internal/aibom/types.go` | Hand-written Go types: `AiBom`, `ModelComponent`, `AgentComponent`, `Guardrail`, `KnowledgeBase`, `ComputeAIAccess`, `AIDependency`, etc. |
| `internal/aibom/cnspec_aibom.proto` | Proto definitions (not yet compiled — types.go used directly) |
| `internal/aibom/pack/aibom.mql.yaml` | Embedded AIBOM query pack with per-platform queries |
| `internal/aibom/pack/pack.go` | Query pack embedding and loading |
| `internal/aibom/generator/generator.go` | Parse MQL report into AIBOM components per provider; AI dependency classification; compute AI access detection |
| `internal/aibom/generator/report_collection.go` | `AiBomFields` struct for deserializing MQL report data points |
| `internal/aibom/generator/scoring.go` | Completeness scoring algorithm |
| `internal/aibom/cyclonedx.go` | CycloneDX ML-BOM rendering with model cards, SPDX license normalization |
| `internal/aibom/json.go` | cnspec-json format rendering |
| `internal/aibom/textlist.go` | Human-readable table format with glamour markdown rendering |
| `internal/aibom/aibom.go` | Format registry and output dispatch |

## MQL Provider Dependencies

The `cnspec aibom` command uses MQL resources from the OS provider and cloud providers. All listed providers have the required resources implemented:

| Provider | MQL Resources |
|----------|--------------|
| OS (local/ssh/container) | `ai.models`, 25 coding agent resources (`claude.code`, `cursor`, `github.copilot`, etc.), `npm.packages`, `python.packages` |
| AWS | `aws.bedrock.foundationModels`, `aws.bedrock.customModels`, `aws.bedrock.agents`, `aws.bedrock.knowledgeBases`, `aws.bedrock.guardrails`, `aws.bedrock.flows`, `aws.sagemaker.models`, `aws.lambda.functions` |
| GCP | `gcp.project.vertexaiService.models`, `gcp.project.modelArmorService.templates`, `gcp.project.cloudFunctions`, `gcp.project.cloudRunService.services`, `gcp.project.iamPolicy` |
| Azure | `azure.subscription.cognitiveServices().accounts` (incl. `raiPolicies`), `azure.subscription.functionsService.functions` |
| Ollama | `ollama.models` (with `info` block and license detection) |
| HuggingFace | `huggingface.models` (with `config`, `cardData`) |
| OpenAI | `openai.models`, `openai.fineTuningJobs`, `openai.vectorStores`, `openai.projects` |
| Claude (Anthropic) | `claude.models`, `claude.agents`, `claude.skills`, `claude.environments`, `claude.vaults`, `claude.memoryStores` |
| vLLM | `vllm.server`, `vllm.models` |

## Consequences

**Advantages:**
- Dedicated `cnspec aibom` command — clear separation from `cnspec sbom`, aligned with [CycloneDX BOM taxonomy](https://cyclonedx.org/capabilities/)
- Full AI supply chain visibility — models, agents, guardrails, knowledge bases, compute access, and AI dependencies in a single BOM
- Continuous & policy-driven — unlike one-shot generators, cnspec can enforce completeness thresholds and security baselines across fleets of models
- Multi-cloud + local — single tool covers cloud platforms (AWS, GCP, Azure), inference APIs (OpenAI, Anthropic), model registries (HuggingFace, Ollama), local runtimes (vLLM), and local coding agents (25 types)
- Standards-based — [CycloneDX ML-BOM](https://cyclonedx.org/capabilities/mlbom/) output interoperates with existing BOM tooling
- Shadow AI detection — local agent discovery (25 coding agents with MCP/plugins/skills), cached model detection (8 filesystem detectors), and AI dependency classification provide visibility into unauthorized AI adoption

**Risks:**
- **Metadata asymmetry:** HuggingFace and Ollama completeness scores will be higher than cloud providers that expose less metadata. Scores reflect provider transparency, not security posture.
- **Agent detection lag:** New AI coding agents appear frequently. The detection list must be maintained.
- **AI library registry maintenance:** The `aiPackageRegistry` (~80 packages) needs periodic updates as the AI/ML ecosystem evolves.

## References

- [CycloneDX ML-BOM Capability](https://cyclonedx.org/capabilities/mlbom/)
- [CycloneDX 1.6 JSON Schema — modelCard](https://cyclonedx.org/docs/1.6/json/#components_items_modelCard)
- [CycloneDX 1.6 JSON Schema — component types](https://cyclonedx.org/docs/1.6/json/#components_items_type)
- [CycloneDX 1.6 JSON Schema — data](https://cyclonedx.org/docs/1.6/json/#components_items_data)
- [CycloneDX Specification (ECMA-424)](https://ecma-international.org/publications-and-standards/standards/ecma-424/)
- [cyclonedx-go library](https://github.com/CycloneDX/cyclonedx-go)
