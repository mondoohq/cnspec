# ADR-0003: AI Bill of Materials (AIBOM) Generation

**Date:** 2026-05-22
**Status:** Proposed

## Context

Organizations deploying AI models across multiple clouds lack a unified inventory of what models they run, where they came from, what data they were trained on, and what licenses or ethical constraints apply. The OWASP AIBOM initiative and CycloneDX ML-BOM standard (CycloneDX 1.5+) formalize this as an **AI Bill of Materials** — a machine-readable inventory of ML models with provenance, training data, intended use, limitations, and risk metadata.

cnspec already generates software SBOMs via `cnspec sbom` in CycloneDX, SPDX, and custom JSON formats. The CycloneDX Go library we depend on (v0.11.0) already includes full ML-BOM support:

- `ComponentTypeMachineLearningModel` and `ComponentTypeData` component types
- `MLModelCard` with `MLModelParameters`, `MLQuantitativeAnalysis`, `MLModelCardConsiderations`
- `MLModelCardEthicalConsideration`, `MLModelCardFairnessAssessment`

No library upgrade is needed.

Existing cloud providers already expose significant AI/ML resources:

| Provider | Existing MQL Resources | Metadata Richness |
|----------|----------------------|-------------------|
| AWS Bedrock | `aws.bedrock.foundationModels`, `customModels`, `guardrails` | Medium — modalities, lifecycle, provider name |
| AWS SageMaker | `aws.sagemaker.models`, `modelCards`, `trainingjob`, `modelPackages` | High when model cards are populated |
| GCP Vertex AI | `gcp.vertexai.models`, `datasets`, `endpoints`, `pipelineJobs` | Medium — model source info, versioning |
| Azure | No AI/ML resources | None — new resources required |
| HuggingFace | No provider exists | Highest potential via model cards |

## Decision

### 1. New `cnspec aibom` command

Add a new top-level `cnspec aibom` command, parallel to the existing `cnspec sbom`:

```bash
cnspec sbom local -o cyclonedx-json               # software BOM (unchanged)

cnspec aibom aws -o cyclonedx-json                 # AI BOM from AWS (Bedrock + SageMaker)
cnspec aibom gcp -o cyclonedx-json                 # AI BOM from GCP (Vertex AI)
cnspec aibom huggingface -o cyclonedx-json         # AI BOM from HuggingFace
cnspec aibom azure -o cyclonedx-json               # AI BOM from Azure (OpenAI + ML)
cnspec aibom ollama -o cyclonedx-json              # AI BOM from local Ollama
cnspec aibom nvidia -o cyclonedx-json              # AI BOM from NVIDIA NGC
cnspec aibom databricks -o cyclonedx-json          # AI BOM from Databricks
cnspec aibom mlflow -o cyclonedx-json              # AI BOM from MLflow
```

**Rationale:** CycloneDX defines BOM as an umbrella standard with SBOM, ML-BOM, CBOM, HBOM, etc. as peer types. AIBOM is the OWASP term for AI-specific Bills of Materials covering models, datasets, training pipelines, ethical considerations, and provenance. A dedicated `cnspec aibom` command keeps the naming clear and avoids overloading `cnspec sbom` with `--type` flags that change its fundamental behavior. The two commands share format rendering infrastructure (`cyclonedx.go`, `spdx.go`) but have separate query packs and generators.

### 2. Extend the SBOM data model with ML-specific messages

The existing `Package` message stays for software components. A new `ModelComponent` message captures AI-specific fields alongside it in the `Sbom` proto:

```protobuf
message Sbom {
  // ... existing fields 1-6 (generator, timestamp, status, asset, packages, error_message) ...
  repeated ModelComponent models = 7;
  CompletenessScore completeness = 8;
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

Each `ModelComponent` maps to a `cyclonedx.Component` with `Type: ComponentTypeMachineLearningModel` and a populated `ModelCard` field:

- `ModelCard.ModelParameters` — task, architectureFamily, modelArchitecture, approach, datasets, inputs/outputs
- `ModelCard.Considerations` — useCases (from intended_uses), technicalLimitations, ethicalConsiderations
- `ModelCard.QuantitativeAnalysis` — performance metrics as `PerformanceMetric` entries
- `Component.Licenses` from model license
- `Component.PackageURL` with provider-specific PURL
- `Component.ExternalReferences` for source URL, model card URL

Training datasets render as nested `ComponentTypeData` sub-components.

### 4. Completeness scoring (like OWASP AIBOM Generator)

Weighted scoring per model, averaged across the BOM:

| Section | Weight | Fields |
|---------|--------|--------|
| Identity | 0.25 | name, version, provider, model_id, author |
| License & Legal | 0.20 | license |
| Technical | 0.20 | task, architecture, approach, input/output modalities |
| Training & Data | 0.20 | training_datasets, provenance |
| Ethics & Risk | 0.15 | ethical_considerations, limitations, intended_uses |

Output as CycloneDX BOM-level properties and in cnspec-json as the `completeness` field.

## MQL Requirements by Provider

### HuggingFace — New Provider Required

**Location:** `mql/providers/huggingface/` (cnquery repo)

**Connection:**
- API token via `HUGGINGFACE_TOKEN` env var or `--token` CLI flag
- REST client against `https://huggingface.co/api/`
- Scope: org/user account level — list all models for the authenticated user/org

**Resources needed:**

```lr
huggingface @defaults("user") {
  // Authenticated user or organization
  user string
  // All models accessible to this account
  models() []huggingface.model
}

huggingface.model @defaults("id author") {
  // Full model ID (e.g., "meta-llama/Llama-3.1-8B")
  id string
  // Model author or organization
  author string
  // Internal model identifier
  modelId string
  // Git SHA of the current revision
  sha string
  // Model tags (e.g., ["pytorch", "transformers", "text-generation"])
  tags []string
  // Pipeline task (e.g., "text-generation", "image-classification")
  pipelineTag string
  // ML library (e.g., "transformers", "diffusers")
  libraryName string
  // License identifier (e.g., "apache-2.0", "llama3.1")
  license string
  // Total downloads
  downloads int
  // Total likes
  likes int
  // Whether the model is private
  private bool
  // Creation time
  createdAt time
  // Last modification time
  lastModified time
  // Parsed YAML frontmatter from model card (datasets, language, metrics, etc.)
  cardData() dict
  // Raw model card content (README.md)
  modelCard() string
  // Parsed config.json (architecture, hidden_size, num_layers, vocab_size, etc.)
  config() dict
  // File listing in the model repository
  siblings() []dict
}
```

**API endpoints:**
- `GET /api/models?author={user}` — list models for user/org
- `GET /api/models/{model_id}` — full model metadata
- `GET /api/models/{model_id}/resolve/main/README.md` — model card
- `GET /api/models/{model_id}/resolve/main/config.json` — model config

**AIBOM field mapping:**

| AIBOM Field | HuggingFace Source |
|-------------|-------------------|
| name | `model.id` |
| version | `model.sha` |
| author | `model.author` |
| license | `model.license` or `cardData["license"]` |
| task | `model.pipelineTag` |
| architecture_family | `config["model_type"]` (e.g., "llama", "gpt2") |
| model_architecture | `config["architectures"][0]` (e.g., "LlamaForCausalLM") |
| training_datasets | `cardData["datasets"]` |
| intended_uses | Parsed from model card "Intended Use" section |
| limitations | Parsed from model card "Limitations" section |
| performance_metrics | `cardData["model-index"][*]["results"]` |
| input_modalities | Derived from `pipelineTag` (text-generation → ["text"]) |
| output_modalities | Derived from `pipelineTag` |

**PURL format:** `pkg:huggingface/{author}/{model-name}@{sha}`

**Expected completeness: HIGH** — HuggingFace model cards provide the richest metadata of any provider.

### AWS Bedrock — Existing Resources, Query Pack Only

**No new MQL resources needed.** Existing `aws.bedrock.foundationModels` and `aws.bedrock.customModels` cover the required fields.

**AIBOM query pack MQL:**

```mql
aws.bedrock.foundationModels {
  modelId
  modelName
  providerName
  modelArn
  inputModalities
  outputModalities
  customizationsSupported
  inferenceTypesSupported
  responseStreamingSupported
  modelLifecycleStatus
}

aws.bedrock.customModels {
  modelArn
  modelName
  customizationType
  baseModel { modelId modelName providerName }
  trainingDataConfig
}
```

**AIBOM field mapping:**

| AIBOM Field | Bedrock Source |
|-------------|--------------|
| name | `foundationModel.modelName` |
| model_id | `foundationModel.modelId` |
| author | `foundationModel.providerName` |
| input_modalities | `foundationModel.inputModalities` |
| output_modalities | `foundationModel.outputModalities` |
| provenance | `customModel.baseModel`, `customModel.trainingDataConfig` |

**Fields NOT available from Bedrock:** license, task, architecture, training_datasets, intended_uses, limitations, ethical_considerations, performance_metrics.

**PURL format:** `pkg:aws/bedrock/{modelId}`

**Expected completeness: LOW-MEDIUM**

### AWS SageMaker — Existing Resources, Query Pack Only

**No new MQL resources needed.** Existing `aws.sagemaker.models`, `aws.sagemaker.modelCards`, and `aws.sagemaker.trainingjob` cover the required fields.

**AIBOM query pack MQL:**

```mql
aws.sagemaker.modelCards {
  name
  arn
  modelCardStatus
  content
  createdAt
  lastModifiedAt
  tags
}

aws.sagemaker.models {
  name
  arn
  createdAt
  primaryContainer
  tags
}
```

**AIBOM field mapping:** The `modelCard.content()` returns a JSON string with structured sections. The AIBOM generator must parse this JSON:

| AIBOM Field | SageMaker Model Card JSON Path |
|-------------|-------------------------------|
| name | `model_overview.model_name` |
| description | `model_overview.model_description` |
| author | `model_overview.model_owner` |
| version | `model_overview.model_version` |
| task | `model_overview.problem_type` |
| intended_uses | `intended_uses.purpose_of_model`, `intended_uses.intended_uses` |
| limitations | `intended_uses.factors_affecting_model_efficiency`, `intended_uses.risk_rating` |
| training_datasets | `training_details.training_observations`, `training_details.training_job_details` |
| ethical_considerations | `ethical_considerations[*]` |
| performance_metrics | `evaluation_details[*].metric_groups[*]` |

**PURL format:** `pkg:aws/sagemaker/{account-id}/{model-name}`

**Expected completeness: MEDIUM-HIGH** (when model cards are filled out; many are sparse)

### GCP Vertex AI — Existing Resources, Query Pack Only

**No new MQL resources needed.** Existing `gcp.vertexai.models`, `gcp.vertexai.datasets`, and `gcp.vertexai.endpoints` cover the required fields.

**AIBOM query pack MQL:**

```mql
gcp.vertexai.models {
  name
  displayName
  description
  versionId
  versionAliases
  modelSourceInfo
  containerSpec
  supportedDeploymentResourcesTypes
  supportedInputStorageFormats
  supportedOutputStorageFormats
}

gcp.vertexai.datasets {
  name
  displayName
  metadataSchemaUri
}
```

**AIBOM field mapping:**

| AIBOM Field | Vertex AI Source |
|-------------|-----------------|
| name | `model.displayName` |
| version | `model.versionId` |
| description | `model.description` |
| provenance.source_type | `model.modelSourceInfo["sourceType"]` (AUTOML, CUSTOM, MODEL_GARDEN) |
| input_modalities | Derived from `supportedInputStorageFormats` |
| output_modalities | Derived from `supportedOutputStorageFormats` |
| training_datasets | Correlated from `datasets` via pipeline jobs |

**Fields NOT available from Vertex AI:** license, author, task, architecture, intended_uses, limitations, ethical_considerations.

**PURL format:** `pkg:gcp/vertexai/{project-id}/{model-name}@{version-id}`

**Expected completeness: LOW-MEDIUM**

### Azure OpenAI + Azure ML — New Resources Required

**Location:** Azure provider in cnquery repo (`mql/providers/azure/`)

**New resources needed in `azure.lr`:**

```lr
// Azure OpenAI (Cognitive Services accounts with kind=OpenAI)
azure.subscription.openaiService {
  // Subscription ID
  subscriptionId string
  // OpenAI accounts (Cognitive Services with kind=OpenAI)
  accounts() []azure.subscription.openaiService.account
}

azure.subscription.openaiService.account @defaults("name location") {
  // Resource ID
  id string
  // Account name
  name string
  // Azure region
  location string
  // SKU tier
  sku dict
  // Model deployments
  deployments() []azure.subscription.openaiService.deployment
}

azure.subscription.openaiService.deployment @defaults("name modelName modelVersion") {
  // Deployment ID
  id string
  // Deployment name
  name string
  // Deployed model name (e.g., "gpt-4o", "text-embedding-ada-002")
  modelName string
  // Model version
  modelVersion string
  // Model format (e.g., "OpenAI")
  modelFormat string
  // Scale type (e.g., "Standard", "Provisioned-Managed")
  scaleType string
  // Provisioned capacity units
  capacity int
  // Deployment status
  provisioningState string
}

// Azure Machine Learning
azure.subscription.mlService {
  subscriptionId string
  workspaces() []azure.subscription.mlService.workspace
}

azure.subscription.mlService.workspace @defaults("name location") {
  id string
  name string
  location string
  // Registered models
  models() []azure.subscription.mlService.model
}

azure.subscription.mlService.model @defaults("name version") {
  id string
  name string
  version string
  description string
  tags map[string]string
  properties map[string]string
  // Model URI (azureml://..., runs://...)
  modelUri string
  // Model type (custom, mlflow, triton)
  modelType string
}
```

**Azure SDK dependencies needed:**
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cognitiveservices/armcognitiveservices` — for OpenAI accounts and deployments
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/machinelearning/armmachinelearning` — for ML workspaces and models

**AIBOM field mapping:**

| AIBOM Field | Azure Source |
|-------------|-------------|
| name | `deployment.modelName` or `mlModel.name` |
| version | `deployment.modelVersion` or `mlModel.version` |
| model_id | `deployment.id` or `mlModel.id` |
| description | `mlModel.description` |
| provenance | `mlModel.modelUri`, `mlModel.modelType` |

**Fields NOT available from Azure:** license, author, task, architecture, training_datasets, intended_uses, limitations, ethical_considerations, performance_metrics.

**PURL format:** `pkg:azure/openai/{account-name}/{deployment-name}@{model-version}`

**Expected completeness: LOW**

### NVIDIA NGC — New Provider Required

**Location:** `mql/providers/nvidia/` (cnquery repo)

**Connection:**
- API key via `NGC_API_KEY` env var or `--token` flag (key prefixed `nvapi-`)
- REST client against `https://api.ngc.nvidia.com/v2/`
- Scope: org-level — list all models in the authenticated org

**Resources needed:**

```lr
nvidia @defaults("org") {
  // NGC organization
  org string
  // Models in the NGC catalog for this org
  models() []nvidia.model
}

nvidia.model @defaults("name displayName publisher") {
  // Model name (registry identifier)
  name string
  // Human-readable display name
  displayName string
  // Short description
  shortDescription string
  // Full description
  description string
  // Publisher (e.g., "nvidia", "meta", "google")
  publisher string
  // ML framework (e.g., "TensorRT", "PyTorch", "ONNX")
  framework string
  // Precision (e.g., "FP32", "FP16", "INT8")
  precision string
  // Application domain (e.g., "Object Detection", "Text Generation")
  application string
  // Model format (e.g., "ONNX", "TensorRT", "SafeTensors")
  format string
  // Access type (e.g., "LISTED", "UNLISTED")
  accessType string
  // Creation time
  createdAt time
  // Last update time
  updatedAt time
}
```

**API endpoints:**
- `GET /v2/org/{org}/models` — list models in org
- `GET /v2/search/catalog/resources?q={query}&type=MODEL` — search public catalog
- `GET /v2/org/{org}/models/{model}/versions/{version}/files/` — model files

**AIBOM field mapping:**

| AIBOM Field | NGC Source |
|-------------|-----------|
| name | `model.displayName` |
| model_id | `model.name` |
| author | `model.publisher` |
| description | `model.description` |
| task | `model.application` |
| architecture_family | `model.framework` |
| provenance.format | `model.format` |
| provenance.precision | `model.precision` |

**Fields NOT available from NGC:** license, training_datasets, intended_uses, limitations, ethical_considerations, performance_metrics.

**PURL format:** `pkg:nvidia/{org}/{model-name}@{version}`

**Expected completeness: LOW-MEDIUM** — NGC has model identity and technical metadata but lacks training data, ethics, and licensing info.

### Ollama — New Provider Required

**Location:** `mql/providers/ollama/` (cnquery repo)

**Connection:**
- REST client against `http://localhost:11434` (configurable via `--host` flag or `OLLAMA_HOST` env var)
- No authentication required for local access
- Scope: instance-level — list all locally installed models

**Resources needed:**

```lr
ollama @defaults("host") {
  // Ollama server host
  host string
  // All locally installed models
  models() []ollama.model
}

ollama.model @defaults("name family parameterSize") {
  // Model name (e.g., "gemma3", "llama3.1:70b")
  name string
  // Full model identifier
  model string
  // Last modification time
  modifiedAt time
  // Model size in bytes
  size int
  // SHA256 digest
  digest string
  // Model format (e.g., "gguf")
  format string
  // Model family (e.g., "gemma", "llama")
  family string
  // All model families
  families []string
  // Parameter size (e.g., "4.3B", "70B")
  parameterSize string
  // Quantization level (e.g., "Q4_K_M", "F16")
  quantizationLevel string
  // License text (from /api/show)
  license string
  // System prompt
  system string
  // Modelfile contents
  modelfile string
  // Template for prompt formatting
  template string
  // Model parameters (stop tokens, temperature, etc.)
  parameters dict
  // Parent model (if fine-tuned)
  parentModel string
}
```

**API endpoints:**
- `GET /api/tags` — list all installed models (name, size, digest, details)
- `POST /api/show` — detailed model info (license, modelfile, template, parameters, system)

**AIBOM field mapping:**

| AIBOM Field | Ollama Source |
|-------------|-------------|
| name | `model.name` |
| version | `model.digest` (SHA256) |
| architecture_family | `model.family` |
| license | `model.license` (from /api/show) |
| provenance.format | `model.format` |
| provenance.quantization | `model.quantizationLevel` |
| provenance.parameter_size | `model.parameterSize` |
| provenance.parent_model | `model.parentModel` |

**Fields NOT available from Ollama:** author, task, training_datasets, intended_uses, limitations, ethical_considerations, performance_metrics, input/output modalities.

**PURL format:** `pkg:ollama/{model-name}@{digest}`

**Expected completeness: LOW** — Ollama has model identity, family, quantization, and license but no training data or ethics info.

### Databricks Unity Catalog — New Provider Required

**Location:** `mql/providers/databricks/` (cnquery repo)

**Connection:**
- Personal access token via `DATABRICKS_TOKEN` env var or `--token` flag
- Workspace URL via `DATABRICKS_HOST` env var or `--host` flag
- REST client against `https://<workspace>.cloud.databricks.com/api/2.1/unity-catalog/`
- Scope: workspace-level — list all registered models

**Resources needed:**

```lr
databricks @defaults("host") {
  // Databricks workspace host
  host string
  // Registered models in Unity Catalog
  registeredModels() []databricks.registeredModel
}

databricks.registeredModel @defaults("fullName owner") {
  // Model name
  name string
  // Catalog name
  catalogName string
  // Schema name
  schemaName string
  // Full three-level name (catalog.schema.model)
  fullName string
  // Model owner
  owner string
  // Model description/comment
  comment string
  // Model aliases (e.g., "champion", "challenger")
  aliases []dict
  // Storage location
  storageLocation string
  // Creation time
  createdAt time
  // Creator user
  createdBy string
  // Last update time
  updatedAt time
  // Last updater user
  updatedBy string
  // Model versions
  versions() []databricks.modelVersion
}

databricks.modelVersion @defaults("version status") {
  // Version number
  version int
  // Model full name
  modelName string
  // Source URI (e.g., dbfs:/..., s3://.., runs:/..)
  source string
  // MLflow run ID (links to training run with params/metrics)
  runId string
  // Version status
  status string
  // Description/comment
  comment string
  // Version aliases
  aliases []string
  // Storage location
  storageLocation string
  // Creation time
  createdAt time
  // Creator user
  createdBy string
  // Last update time
  updatedAt time
}
```

**API endpoints:**
- `GET /api/2.1/unity-catalog/registered-models` — list models (filterable by catalog/schema)
- `GET /api/2.1/unity-catalog/registered-models/{full_name}` — get model details
- `GET /api/2.1/unity-catalog/models/{full_name}/versions` — list versions
- `GET /api/2.1/unity-catalog/models/{full_name}/versions/{version}` — version details

**AIBOM field mapping:**

| AIBOM Field | Databricks Source |
|-------------|------------------|
| name | `registeredModel.name` |
| version | `modelVersion.version` |
| author | `registeredModel.owner` or `registeredModel.createdBy` |
| description | `registeredModel.comment` |
| provenance.source | `modelVersion.source` |
| provenance.run_id | `modelVersion.runId` (links to MLflow run for params/metrics) |

**Fields NOT available from Databricks model registry directly:** license, task, architecture, training_datasets, intended_uses, limitations, ethical_considerations. Training metrics/params are available indirectly via `runId` → MLflow Runs API.

**PURL format:** `pkg:databricks/{catalog}/{schema}/{model-name}@{version}`

**Expected completeness: LOW-MEDIUM** — Identity and provenance are good; training details require following `runId` to the MLflow runs API.

### MLflow — New Provider Required

**Location:** `mql/providers/mlflow/` (cnquery repo)

**Connection:**
- REST client against `http://<tracking-server>:5000/api/` (configurable via `--host` flag or `MLFLOW_TRACKING_URI` env var)
- Authentication depends on deployment (token, basic auth, or none for local)
- Scope: server-level — list all registered models

**Resources needed:**

```lr
mlflow @defaults("host") {
  // MLflow tracking server host
  host string
  // Registered models
  registeredModels() []mlflow.registeredModel
}

mlflow.registeredModel @defaults("name") {
  // Model name
  name string
  // Description
  description string
  // Tags (key-value)
  tags map[string]string
  // Aliases (e.g., "champion", "production")
  aliases []dict
  // Creation time
  createdAt time
  // Last update time
  updatedAt time
  // Creator user ID
  userId string
  // Latest versions
  latestVersions() []mlflow.modelVersion
  // All versions
  versions() []mlflow.modelVersion
}

mlflow.modelVersion @defaults("name version currentStage") {
  // Model name
  name string
  // Version number
  version string
  // Current stage (None, Staging, Production, Archived)
  currentStage string
  // Description
  description string
  // Source artifact URI
  source string
  // MLflow run ID (links to training run)
  runId string
  // Tags (key-value)
  tags map[string]string
  // Aliases
  aliases []string
  // Version status
  status string
  // Creation time
  createdAt time
  // Last update time
  updatedAt time
  // Creator user ID
  userId string
  // Training run details (params, metrics, artifacts)
  run() mlflow.run
}

mlflow.run @defaults("runId status") {
  // Run ID
  runId string
  // Experiment ID
  experimentId string
  // Run status (RUNNING, SCHEDULED, FINISHED, FAILED, KILLED)
  status string
  // Start time
  startTime time
  // End time
  endTime time
  // Run parameters (hyperparameters)
  params map[string]string
  // Run metrics (training/eval metrics)
  metrics map[string]float
  // Tags
  tags map[string]string
  // Artifact URI
  artifactUri string
}
```

**API endpoints:**
- `GET /api/2.0/mlflow/registered-models/search` — list/search models
- `GET /api/2.0/mlflow/registered-models/get?name={name}` — get model
- `GET /api/2.0/mlflow/model-versions/search?filter=name='{name}'` — list versions
- `GET /api/2.0/mlflow/runs/get?run_id={id}` — get training run (params, metrics)

**AIBOM field mapping:**

| AIBOM Field | MLflow Source |
|-------------|-------------|
| name | `registeredModel.name` |
| version | `modelVersion.version` |
| author | `registeredModel.userId` |
| description | `registeredModel.description` |
| provenance.source | `modelVersion.source` |
| provenance.run_id | `modelVersion.runId` |
| performance_metrics | `run.metrics` (via runId) |
| labels | `registeredModel.tags` + `modelVersion.tags` |

**Fields NOT available from MLflow:** license, task (unless tagged), architecture, training_datasets (unless in artifacts), intended_uses, limitations, ethical_considerations.

**PURL format:** `pkg:mlflow/{model-name}@{version}`

**Expected completeness: LOW-MEDIUM** — Model identity and training metrics are available via run linkage; everything else depends on user-applied tags and conventions.

## AIBOM Query Pack

The AIBOM query pack is a **separate, standalone query pack** — not embedded in the SBOM internals. It lives in `content/` alongside other policy and query pack bundles so it can be distributed, versioned, linted, and customized independently.

**File:** `content/mondoo-aibom.mql.yaml`

The software SBOM uses an embedded internal query pack (`internal/sbom/pack/sbom.mql.yaml`) because it targets OS-level packages with a fixed set of queries. The AIBOM query pack is different:

- **Multi-provider:** Queries are filtered by platform (`asset.platform`), so only relevant queries execute per scan target
- **Evolving:** New providers and fields will be added over time; a standalone pack is easier to update
- **Customizable:** Organizations may want to add custom queries or disable specific providers
- **Lintable:** `cnspec policy lint` works on `content/` bundles

**Structure:**

```yaml
packs:
  - uid: mondoo-aibom
    name: Mondoo AI Bill of Materials
    version: 1.0.0
    authors:
      - name: Mondoo, Inc.
    tags:
      mondoo.com/category: security
    queries:
      # --- HuggingFace ---
      - uid: mondoo-aibom-huggingface-models
        title: Retrieve HuggingFace model inventory
        filters:
          - mql: asset.platform == "huggingface"
        mql: |
          huggingface.models {
            id author modelId license pipelineTag libraryName
            downloads likes private createdAt lastModified
            tags cardData config
          }

      # --- AWS Bedrock ---
      - uid: mondoo-aibom-aws-bedrock-foundation-models
        title: Retrieve AWS Bedrock foundation models
        filters:
          - mql: asset.platform == "aws"
        mql: |
          aws.bedrock.foundationModels {
            modelId modelName providerName modelArn
            inputModalities outputModalities
            customizationsSupported inferenceTypesSupported
            responseStreamingSupported modelLifecycleStatus
          }

      - uid: mondoo-aibom-aws-bedrock-custom-models
        title: Retrieve AWS Bedrock custom models
        filters:
          - mql: asset.platform == "aws"
        mql: |
          aws.bedrock.customModels {
            modelArn modelName customizationType
            baseModel { modelId modelName providerName }
            trainingDataConfig outputDataConfig
          }

      # --- AWS SageMaker ---
      - uid: mondoo-aibom-aws-sagemaker-model-cards
        title: Retrieve AWS SageMaker model cards
        filters:
          - mql: asset.platform == "aws"
        mql: |
          aws.sagemaker.modelCards {
            name arn modelCardStatus content
            createdAt lastModifiedAt tags
          }

      - uid: mondoo-aibom-aws-sagemaker-models
        title: Retrieve AWS SageMaker models
        filters:
          - mql: asset.platform == "aws"
        mql: |
          aws.sagemaker.models {
            name arn createdAt primaryContainer tags
          }

      # --- GCP Vertex AI ---
      - uid: mondoo-aibom-gcp-vertexai-models
        title: Retrieve GCP Vertex AI models
        filters:
          - mql: asset.platform == "gcp"
        mql: |
          gcp.vertexai.models {
            name displayName description versionId
            versionAliases modelSourceInfo containerSpec
            supportedDeploymentResourcesTypes
            supportedInputStorageFormats
            supportedOutputStorageFormats
          }

      - uid: mondoo-aibom-gcp-vertexai-datasets
        title: Retrieve GCP Vertex AI datasets
        filters:
          - mql: asset.platform == "gcp"
        mql: |
          gcp.vertexai.datasets {
            name displayName metadataSchemaUri
          }

      # --- Azure OpenAI (Phase 4) ---
      - uid: mondoo-aibom-azure-openai-deployments
        title: Retrieve Azure OpenAI deployments
        filters:
          - mql: asset.platform == "azure"
        mql: |
          azure.subscription.openaiService.accounts {
            name location
            deployments {
              name modelName modelVersion modelFormat
              scaleType capacity provisioningState
            }
          }

      # --- Azure ML (Phase 4) ---
      - uid: mondoo-aibom-azure-ml-models
        title: Retrieve Azure ML registered models
        filters:
          - mql: asset.platform == "azure"
        mql: |
          azure.subscription.mlService.workspaces {
            name location
            models {
              name version description modelUri modelType
              tags properties
            }
          }

      # --- NVIDIA NGC (Phase 5) ---
      - uid: mondoo-aibom-nvidia-models
        title: Retrieve NVIDIA NGC model catalog
        filters:
          - mql: asset.platform == "nvidia"
        mql: |
          nvidia.models {
            name displayName publisher framework
            precision application format
            description createdAt updatedAt
          }

      # --- Ollama (Phase 2) ---
      - uid: mondoo-aibom-ollama-models
        title: Retrieve locally installed Ollama models
        filters:
          - mql: asset.platform == "ollama"
        mql: |
          ollama.models {
            name model digest size modifiedAt
            format family families parameterSize
            quantizationLevel license parentModel
          }

      # --- Databricks (Phase 6) ---
      - uid: mondoo-aibom-databricks-models
        title: Retrieve Databricks Unity Catalog registered models
        filters:
          - mql: asset.platform == "databricks"
        mql: |
          databricks.registeredModels {
            name fullName owner comment
            createdAt createdBy updatedAt updatedBy
            versions {
              version source runId status comment
              createdAt createdBy
            }
          }

      # --- MLflow (Phase 6) ---
      - uid: mondoo-aibom-mlflow-models
        title: Retrieve MLflow registered models
        filters:
          - mql: asset.platform == "mlflow"
        mql: |
          mlflow.registeredModels {
            name description tags userId
            createdAt updatedAt
            latestVersions {
              version currentStage source runId
              description tags status
              run { params metrics tags }
            }
          }
```

**Loading the pack:** The new `aibom.go` command loads the AIBOM query pack from `content/` using `policy.BundleFromPaths()` (same as `cnspec scan` loads policy bundles), separate from the software SBOM's embedded `pack.QueryPack()`.

## cnspec Changes Required

| File | Change |
|------|--------|
| `apps/cnspec/cmd/aibom.go` | New: `cnspec aibom` command, loads `content/mondoo-aibom.mql.yaml` |
| `content/mondoo-aibom.mql.yaml` | New: standalone AIBOM query pack with per-provider queries |
| `internal/sbom/cnspec_sbom.proto` | Add `ModelComponent`, `EthicalConsideration`, `CompletenessScore` messages |
| `mql/sbom/mql_sbom.proto` | Same proto changes (shared definition) |
| `internal/sbom/cyclonedx.go` | Add ML-BOM component rendering (`ComponentTypeMachineLearningModel`, `MLModelCard`) |
| `internal/sbom/generator/aibom_generator.go` | New: parse MQL report into `ModelComponent` protos per provider |
| `internal/sbom/scoring.go` | New: completeness scoring algorithm |

## New MQL Providers Summary

| Provider | Repo | Authentication | Scope | Priority |
|----------|------|---------------|-------|----------|
| **HuggingFace** | cnquery | API token (`HUGGINGFACE_TOKEN`) | Org/user models | Phase 1 |
| **Ollama** | cnquery | None (local) | Instance models | Phase 2 |
| **NVIDIA NGC** | cnquery | API key (`NGC_API_KEY`, `nvapi-` prefix) | Org catalog | Phase 5 |
| **Databricks** | cnquery | PAT (`DATABRICKS_TOKEN`) + workspace URL | Workspace models | Phase 6 |
| **MLflow** | cnquery | Configurable (token/basic/none) | Server models | Phase 6 |

Existing providers requiring new resources:

| Provider | New Resources | Priority |
|----------|--------------|----------|
| **Azure** | `azure.subscription.openaiService.*`, `azure.subscription.mlService.*` | Phase 4 |

Existing providers with no changes needed:

| Provider | Existing Resources Used |
|----------|------------------------|
| **AWS** | `aws.bedrock.foundationModels`, `aws.bedrock.customModels`, `aws.sagemaker.models`, `aws.sagemaker.modelCards` |
| **GCP** | `gcp.vertexai.models`, `gcp.vertexai.datasets`, `gcp.vertexai.endpoints` |

## Phased Rollout

### Phase 1: Foundation + HuggingFace
- Proto extension, `cnspec aibom` command, AIBOM generator, CycloneDX ML-BOM rendering, completeness scoring
- **MQL dependency:** HuggingFace provider (new)

### Phase 2: AWS (Bedrock + SageMaker) + Ollama
- AWS queries in AIBOM query pack, SageMaker model card JSON parsing in generator
- Ollama for local model inventory (shadow AI detection use case)
- **MQL dependency:** Ollama provider (new); AWS uses existing resources

### Phase 3: GCP Vertex AI
- GCP queries in AIBOM query pack, Vertex AI model/dataset mapping in generator
- **MQL dependency:** None — uses existing resources

### Phase 4: Azure OpenAI + Azure ML
- Azure queries in AIBOM query pack, Azure mapping in generator
- **MQL dependency:** New Azure resources (`azure.subscription.openaiService.*`, `azure.subscription.mlService.*`)

### Phase 5: NVIDIA NGC
- NGC catalog queries in AIBOM query pack
- **MQL dependency:** NVIDIA NGC provider (new)

### Phase 6: Databricks + MLflow
- Databricks Unity Catalog and MLflow model registry queries
- MLflow run linkage for training metrics/params
- **MQL dependency:** Databricks provider (new), MLflow provider (new)

### Phase 7: AIBOM Policies
- Completeness enforcement policy (`content/mondoo-aibom-completeness.mql.yaml`)
- AI security checks (model access controls, encryption, lifecycle status, guardrail configuration)

## Consequences

**Advantages:**
- Dedicated `cnspec aibom` command — clear separation from `cnspec sbom`, aligned with CycloneDX BOM taxonomy and OWASP AIBOM naming
- Continuous & policy-driven — unlike one-shot generators, cnspec can enforce completeness thresholds and security baselines across fleets of models
- Multi-cloud + local — single tool covers HuggingFace, AWS, GCP, Azure, NVIDIA NGC, Databricks, MLflow, and Ollama (local)
- Standards-based — CycloneDX ML-BOM output interoperates with existing SBOM tooling
- Shadow AI detection — Ollama provider complements ADR-0002's agent detection with local model inventory

**Risks:**
- **Metadata asymmetry:** HuggingFace completeness scores will be dramatically higher than Azure or Ollama. Need to set expectations that scores reflect provider transparency, not security posture.
- **SageMaker model card parsing:** The `content()` field is a JSON string whose schema varies. Robust parsing with graceful fallbacks is needed.
- **HuggingFace model card parsing:** YAML frontmatter is structured but the markdown body is freeform. Extracting intended_uses/limitations from prose requires heuristic section detection.
- **Azure SDK integration:** Adding new Azure resources requires new SDK dependencies and may expand the Azure provider binary.
- **Five new providers:** HuggingFace, Ollama, NVIDIA NGC, Databricks, and MLflow each require a full provider implementation in cnquery (connection, resources, config, tests). This is the largest effort in the plan.
- **MLflow deployment variability:** MLflow servers range from unauthenticated local instances to Databricks-hosted managed tracking. The provider must handle multiple auth methods gracefully.
- **Ollama availability:** Ollama only exists on hosts where it's installed. The provider must handle connection failures gracefully (host not running).
