// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generator

// Structures to parse the data from cnspec report for AIBOM generation.
// Each provider returns differently shaped JSON; these structs capture
// the union of all fields across Ollama, HuggingFace, AWS, GCP, and Azure.

type AiBomAsset struct {
	Name     string   `json:"name,omitempty"`
	Platform string   `json:"platform,omitempty"`
	Version  string   `json:"version,omitempty"`
	Build    string   `json:"build,omitempty"`
	Family   []string `json:"family,omitempty"`
	Arch     string   `json:"arch,omitempty"`
	IDs      []string `json:"ids,omitempty"`
}

// OllamaModel represents an Ollama model from the MQL report.
type OllamaModel struct {
	Name              string     `json:"name,omitempty"`
	Model             string     `json:"model,omitempty"`
	Family            string     `json:"family,omitempty"`
	Families          []string   `json:"families,omitempty"`
	ParameterSize     string     `json:"parameterSize,omitempty"`
	QuantizationLevel string     `json:"quantizationLevel,omitempty"`
	Format            string     `json:"format,omitempty"`
	Size              int64      `json:"size,omitempty"`
	Digest            string     `json:"digest,omitempty"`
	ModifiedAt        string     `json:"modifiedAt,omitempty"`
	License           string     `json:"license,omitempty"`
	Capabilities      []string   `json:"capabilities,omitempty"`
	Info              OllamaInfo `json:"info"`
}

type OllamaInfo struct {
	Architecture    string   `json:"architecture,omitempty"`
	ParameterCount  int64    `json:"parameterCount,omitempty"`
	ContextLength   int      `json:"contextLength,omitempty"`
	EmbeddingLength int      `json:"embeddingLength,omitempty"`
	BlockCount      int      `json:"blockCount,omitempty"`
	VocabSize       int      `json:"vocabSize,omitempty"`
	ExpertCount     int      `json:"expertCount,omitempty"`
	SizeLabel       string   `json:"sizeLabel,omitempty"`
	Basename        string   `json:"basename,omitempty"`
	Finetune        string   `json:"finetune,omitempty"`
	Author          string   `json:"author,omitempty"`
	Description     string   `json:"description,omitempty"`
	Datasets        []string `json:"datasets,omitempty"`
	Languages       []string `json:"languages,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	License         string   `json:"license,omitempty"`
}

// HuggingFaceModel represents a HuggingFace model from the MQL report.
type HuggingFaceModel struct {
	ID           string         `json:"id,omitempty"`
	ModelID      string         `json:"modelId,omitempty"`
	Author       string         `json:"author,omitempty"`
	PipelineTag  string         `json:"pipelineTag,omitempty"`
	LibraryName  string         `json:"libraryName,omitempty"`
	Tags         []string       `json:"tags,omitempty"`
	Downloads    int            `json:"downloads,omitempty"`
	Likes        int            `json:"likes,omitempty"`
	SHA          string         `json:"sha,omitempty"`
	CreatedAt    string         `json:"createdAt,omitempty"`
	LastModified string         `json:"lastModified,omitempty"`
	Gated        bool           `json:"gated,omitempty"`
	Disabled     bool           `json:"disabled,omitempty"`
	Private      bool           `json:"private,omitempty"`
	License      string         `json:"license,omitempty"`
	CardData     map[string]any `json:"cardData,omitempty"`
	Config       map[string]any `json:"config,omitempty"`
}

// AWSBedrockFoundationModel represents an AWS Bedrock foundation model.
type AWSBedrockFoundationModel struct {
	ModelID                 string   `json:"modelId,omitempty"`
	ModelName               string   `json:"modelName,omitempty"`
	ProviderName            string   `json:"providerName,omitempty"`
	InputModalities         []string `json:"inputModalities,omitempty"`
	OutputModalities        []string `json:"outputModalities,omitempty"`
	InferenceTypesSupported []string `json:"inferenceTypesSupported,omitempty"`
	ModelLifecycleStatus    string   `json:"modelLifecycleStatus,omitempty"`
}

// AWSBedrockCustomModel represents an AWS Bedrock custom/fine-tuned model.
type AWSBedrockCustomModel struct {
	ModelArn  string                     `json:"modelArn,omitempty"`
	ModelName string                     `json:"modelName,omitempty"`
	BaseModel *AWSBedrockFoundationModel `json:"baseModel,omitempty"`
}

// AWSSageMakerModel represents an AWS SageMaker model.
type AWSSageMakerModel struct {
	Name string `json:"name,omitempty"`
	Arn  string `json:"arn,omitempty"`
}

// GCPVertexAIModel represents a GCP Vertex AI model.
type GCPVertexAIModel struct {
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}

// AzureCognitiveServicesAccount represents an Azure AI Services account.
type AzureCognitiveServicesAccount struct {
	ID                  string `json:"id,omitempty"`
	Name                string `json:"name,omitempty"`
	Location            string `json:"location,omitempty"`
	Kind                string `json:"kind,omitempty"`
	Endpoint            string `json:"endpoint,omitempty"`
	PublicNetworkAccess string `json:"publicNetworkAccess,omitempty"`
	DisableLocalAuth    bool   `json:"disableLocalAuth,omitempty"`
}

// VLLMData represents the vLLM server data from the MQL report.
type VLLMData struct {
	Version string      `json:"version,omitempty"`
	Server  *VLLMServer `json:"server,omitempty"`
	Models  []VLLMModel `json:"models,omitempty"`
}

type VLLMServer struct {
	BaseUrl                  string `json:"baseUrl,omitempty"`
	Reachable                bool   `json:"reachable,omitempty"`
	Version                  string `json:"version,omitempty"`
	UsesTls                  bool   `json:"usesTls,omitempty"`
	CorsConfigured           bool   `json:"corsConfigured,omitempty"`
	CorsAllowsAny            bool   `json:"corsAllowsAnyOrigin,omitempty"`
	MetricsExposed           bool   `json:"metricsExposed,omitempty"`
	DocsExposed              bool   `json:"docsExposed,omitempty"`
	OpenapiExposed           bool   `json:"openapiExposed,omitempty"`
	LoadEndpointExposed      bool   `json:"loadEndpointExposed,omitempty"`
	TokenizerInfoExposed     bool   `json:"tokenizerInfoExposed,omitempty"`
	DevEndpointsExposed      bool   `json:"devEndpointsExposed,omitempty"`
	ProfilerEndpointsExposed bool   `json:"profilerEndpointsExposed,omitempty"`
}

type VLLMModel struct {
	ID          string `json:"id,omitempty"`
	Root        string `json:"root,omitempty"`
	Parent      string `json:"parent,omitempty"`
	MaxModelLen int    `json:"maxModelLen,omitempty"`
	Created     string `json:"created,omitempty"`
	OwnedBy     string `json:"ownedBy,omitempty"`
}

// ClaudeModel represents a Claude API model.
type ClaudeModel struct {
	ID                         string `json:"id,omitempty"`
	DisplayName                string `json:"displayName,omitempty"`
	Vendor                     string `json:"vendor,omitempty"`
	Family                     string `json:"family,omitempty"`
	CreatedAt                  string `json:"createdAt,omitempty"`
	MaxInputTokens             int    `json:"maxInputTokens,omitempty"`
	MaxTokens                  int    `json:"maxTokens,omitempty"`
	BatchSupported             bool   `json:"batchSupported,omitempty"`
	CitationsSupported         bool   `json:"citationsSupported,omitempty"`
	CodeExecutionSupported     bool   `json:"codeExecutionSupported,omitempty"`
	ImageInputSupported        bool   `json:"imageInputSupported,omitempty"`
	PdfInputSupported          bool   `json:"pdfInputSupported,omitempty"`
	StructuredOutputsSupported bool   `json:"structuredOutputsSupported,omitempty"`
	ThinkingSupported          bool   `json:"thinkingSupported,omitempty"`
}

// ClaudeAgent represents a Claude managed agent.
type ClaudeAgent struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	System      string `json:"system,omitempty"`
	Model       string `json:"model,omitempty"`
	Version     int    `json:"version,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
}

// ClaudeSkill represents a Claude registered skill.
type ClaudeSkill struct {
	ID            string `json:"id,omitempty"`
	DisplayTitle  string `json:"displayTitle,omitempty"`
	Source        string `json:"source,omitempty"`
	LatestVersion string `json:"latestVersion,omitempty"`
	CreatedAt     string `json:"createdAt,omitempty"`
	UpdatedAt     string `json:"updatedAt,omitempty"`
}

// ClaudeEnvironment represents a Claude execution environment.
type ClaudeEnvironment struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Scope       string `json:"scope,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
}

// ClaudeVault represents a Claude secret vault.
type ClaudeVault struct {
	ID          string             `json:"id,omitempty"`
	DisplayName string             `json:"displayName,omitempty"`
	CreatedAt   string             `json:"createdAt,omitempty"`
	Credentials []ClaudeCredential `json:"credentials,omitempty"`
}

type ClaudeCredential struct {
	ID          string `json:"id,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	VaultId     string `json:"vaultId,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
}

// ClaudeMemoryStore represents a Claude memory store.
type ClaudeMemoryStore struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
}

// OpenAIModel represents an OpenAI API model.
type OpenAIModel struct {
	ID          string `json:"id,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
	OwnedBy     string `json:"ownedBy,omitempty"`
	IsFineTuned bool   `json:"isFineTuned,omitempty"`
	BaseModel   string `json:"baseModel,omitempty"`
}

// OpenAIFineTuningJob represents an OpenAI fine-tuning job.
type OpenAIFineTuningJob struct {
	ID              string         `json:"id,omitempty"`
	Model           string         `json:"model,omitempty"`
	Status          string         `json:"status,omitempty"`
	CreatedAt       string         `json:"createdAt,omitempty"`
	FinishedAt      string         `json:"finishedAt,omitempty"`
	FineTunedModel  string         `json:"fineTunedModel,omitempty"`
	TrainedTokens   int            `json:"trainedTokens,omitempty"`
	OrganizationId  string         `json:"organizationId,omitempty"`
	Hyperparameters map[string]any `json:"hyperparameters,omitempty"`
}

// OpenAIVectorStore represents an OpenAI vector store.
type OpenAIVectorStore struct {
	ID           string         `json:"id,omitempty"`
	Name         string         `json:"name,omitempty"`
	Status       string         `json:"status,omitempty"`
	UsageBytes   int64          `json:"usageBytes,omitempty"`
	CreatedAt    string         `json:"createdAt,omitempty"`
	LastActiveAt string         `json:"lastActiveAt,omitempty"`
	FileCounts   map[string]any `json:"fileCounts,omitempty"`
}

// OpenAIProject represents an OpenAI project.
type OpenAIProject struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Status    string `json:"status,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}

// AWSBedrockGuardrail represents an AWS Bedrock guardrail with policies.
type AWSBedrockGuardrail struct {
	ID                         string         `json:"id,omitempty"`
	Name                       string         `json:"name,omitempty"`
	Arn                        string         `json:"arn,omitempty"`
	Status                     string         `json:"status,omitempty"`
	Version                    string         `json:"version,omitempty"`
	ContentPolicy              map[string]any `json:"contentPolicy,omitempty"`
	SensitiveInformationPolicy map[string]any `json:"sensitiveInformationPolicy,omitempty"`
	TopicPolicy                map[string]any `json:"topicPolicy,omitempty"`
	WordPolicy                 map[string]any `json:"wordPolicy,omitempty"`
}

// AWSBedrockAgent represents an AWS Bedrock agent with dependencies.
type AWSBedrockAgent struct {
	ID              string `json:"id,omitempty"`
	Name            string `json:"name,omitempty"`
	Arn             string `json:"arn,omitempty"`
	Status          string `json:"status,omitempty"`
	Description     string `json:"description,omitempty"`
	FoundationModel string `json:"foundationModel,omitempty"`
	Instruction     string `json:"instruction,omitempty"`
	ActionGroups    []any  `json:"actionGroups,omitempty"`
	KnowledgeBases  []any  `json:"knowledgeBases,omitempty"`
}

// AWSBedrockKnowledgeBase represents an AWS Bedrock knowledge base with data sources.
type AWSBedrockKnowledgeBase struct {
	ID                         string         `json:"id,omitempty"`
	Name                       string         `json:"name,omitempty"`
	Arn                        string         `json:"arn,omitempty"`
	Status                     string         `json:"status,omitempty"`
	Description                string         `json:"description,omitempty"`
	StorageConfiguration       map[string]any `json:"storageConfiguration,omitempty"`
	KnowledgeBaseConfiguration map[string]any `json:"knowledgeBaseConfiguration,omitempty"`
	DataSources                []any          `json:"dataSources,omitempty"`
}

// AWSBedrockFlow represents an AWS Bedrock flow (agent workflow).
type AWSBedrockFlow struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Arn         string `json:"arn,omitempty"`
	Status      string `json:"status,omitempty"`
	Version     string `json:"version,omitempty"`
	Description string `json:"description,omitempty"`
}

// GCPVertexAIPipelineJob represents a GCP Vertex AI pipeline job.
type GCPVertexAIPipelineJob struct {
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	State       string `json:"state,omitempty"`
}

// GCPVertexAIDataset represents a GCP Vertex AI dataset.
type GCPVertexAIDataset struct {
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`
}

// GCPModelArmorTemplate represents a GCP Model Armor safety template.
type GCPModelArmorTemplate struct {
	Name         string            `json:"name,omitempty"`
	ProjectID    string            `json:"projectId,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	FilterConfig map[string]any    `json:"filterConfig,omitempty"`
}

// GCPModelArmorFloorSetting represents the GCP Model Armor floor setting.
type GCPModelArmorFloorSetting struct {
	Name                          string         `json:"name,omitempty"`
	EnableFloorSettingEnforcement bool           `json:"enableFloorSettingEnforcement,omitempty"`
	FilterConfig                  map[string]any `json:"filterConfig,omitempty"`
}

// AzureRAIAccount represents an Azure Cognitive Services account with RAI policies.
type AzureRAIAccount struct {
	Name        string           `json:"name,omitempty"`
	Kind        string           `json:"kind,omitempty"`
	RAIPolicies []AzureRAIPolicy `json:"raiPolicies,omitempty"`
}

type AzureRAIPolicy struct {
	Name           string                  `json:"name,omitempty"`
	Mode           string                  `json:"mode,omitempty"`
	PolicyType     string                  `json:"policyType,omitempty"`
	ContentFilters []AzureRAIContentFilter `json:"contentFilters,omitempty"`
}

type AzureRAIContentFilter struct {
	Name              string `json:"name,omitempty"`
	Source            string `json:"source,omitempty"`
	Enabled           bool   `json:"enabled,omitempty"`
	Blocking          bool   `json:"blocking,omitempty"`
	SeverityThreshold string `json:"severityThreshold,omitempty"`
}

// CodingAgent represents a local coding agent from the OS provider.
type CodingAgent struct {
	ConfigPath       string               `json:"configPath,omitempty"`
	Version          string               `json:"version,omitempty"`
	Email            string               `json:"email,omitempty"`
	Organization     string               `json:"organization,omitempty"`
	Subscription     string               `json:"subscription,omitempty"`
	AuthMode         string               `json:"authMode,omitempty"`
	AuthType         string               `json:"authType,omitempty"`
	Provider         string               `json:"provider,omitempty"`
	Model            string               `json:"model,omitempty"`
	TelemetryEnabled bool                 `json:"telemetryEnabled,omitempty"`
	Skills           []CodingAgentSkill   `json:"skills,omitempty"`
	McpServers       []CodingAgentMcp     `json:"mcpServers,omitempty"`
	Plugins          []CodingAgentPlugin  `json:"plugins,omitempty"`
	Extensions       []CodingAgentExt     `json:"extensions,omitempty"`
	Connectors       []CodingAgentConn    `json:"connectors,omitempty"`
	Accounts         []CodingAgentAccount `json:"accounts,omitempty"`
	ExtensionNames   []string             `json:"extensions_str,omitempty"`
}

type CodingAgentSkill struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source,omitempty"`
	Sha256      string `json:"sha256,omitempty"`
}

type CodingAgentMcp struct {
	Name        string   `json:"name,omitempty"`
	Type        string   `json:"type,omitempty"`
	Command     string   `json:"command,omitempty"`
	Args        []string `json:"args,omitempty"`
	Url         string   `json:"url,omitempty"`
	HasEnv      bool     `json:"hasEnv,omitempty"`
	NeedsAuth   bool     `json:"needsAuth,omitempty"`
	LastChecked string   `json:"lastChecked,omitempty"`
	Plugin      string   `json:"plugin,omitempty"`
}

type CodingAgentPlugin struct {
	Name         string   `json:"name,omitempty"`
	Version      string   `json:"version,omitempty"`
	Description  string   `json:"description,omitempty"`
	Author       string   `json:"author,omitempty"`
	Category     string   `json:"category,omitempty"`
	Scope        string   `json:"scope,omitempty"`
	InstallPath  string   `json:"installPath,omitempty"`
	InstalledAt  string   `json:"installedAt,omitempty"`
	LastUpdated  string   `json:"lastUpdated,omitempty"`
	GitCommitSha string   `json:"gitCommitSha,omitempty"`
	Enabled      bool     `json:"enabled,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	HasMcp       bool     `json:"hasMcp,omitempty"`
	HasHooks     bool     `json:"hasHooks,omitempty"`
}

type CodingAgentExt struct {
	Name        string `json:"name,omitempty"`
	Enabled     bool   `json:"enabled,omitempty"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
	Bundled     bool   `json:"bundled,omitempty"`
	Timeout     int    `json:"timeout,omitempty"`
}

type CodingAgentConn struct {
	Name   string `json:"name,omitempty"`
	ID     string `json:"id,omitempty"`
	Plugin string `json:"plugin,omitempty"`
}

type CodingAgentAccount struct {
	User        string `json:"user,omitempty"`
	GithubAppID string `json:"githubAppId,omitempty"`
}

// AWSLambdaFunction represents an AWS Lambda function with IAM chain.
type AWSLambdaFunction struct {
	Arn              string            `json:"arn,omitempty"`
	Name             string            `json:"name,omitempty"`
	Runtime          string            `json:"runtime,omitempty"`
	Handler          string            `json:"handler,omitempty"`
	CodeSha256       string            `json:"codeSha256,omitempty"`
	CodeSize         int64             `json:"codeSize,omitempty"`
	PackageType      string            `json:"packageType,omitempty"`
	ImageUri         string            `json:"imageUri,omitempty"`
	ResolvedImageUri string            `json:"resolvedImageUri,omitempty"`
	Environment      map[string]string `json:"environment,omitempty"`
	Layers           []AWSLambdaLayer  `json:"layers,omitempty"`
	Role             *AWSIAMRole       `json:"role,omitempty"`
}

type AWSLambdaLayer struct {
	Arn      string `json:"arn,omitempty"`
	CodeSize int64  `json:"codeSize,omitempty"`
}

type AWSIAMRole struct {
	Arn              string         `json:"arn,omitempty"`
	AttachedPolicies []AWSIAMPolicy `json:"attachedPolicies,omitempty"`
}

type AWSIAMPolicy struct {
	Name           string               `json:"name,omitempty"`
	DefaultVersion *AWSIAMPolicyVersion `json:"defaultVersion,omitempty"`
}

type AWSIAMPolicyVersion struct {
	Document map[string]any `json:"document,omitempty"`
}

// GCPCloudFunction represents a GCP Cloud Function with service account.
type GCPCloudFunction struct {
	ProjectID     string                   `json:"projectId,omitempty"`
	Name          string                   `json:"name,omitempty"`
	Description   string                   `json:"description,omitempty"`
	State         string                   `json:"state,omitempty"`
	Environment   string                   `json:"environment,omitempty"`
	URL           string                   `json:"url,omitempty"`
	BuildConfig   *GCPCloudFuncBuildConfig `json:"buildConfig,omitempty"`
	ServiceConfig *GCPCloudFuncSvcConfig   `json:"serviceConfig,omitempty"`
}

type GCPCloudFuncBuildConfig struct {
	Runtime          string `json:"runtime,omitempty"`
	EntryPoint       string `json:"entryPoint,omitempty"`
	DockerRepository string `json:"dockerRepository,omitempty"`
	ServiceAccount   string `json:"serviceAccount,omitempty"`
}

type GCPCloudFuncSvcConfig struct {
	ServiceAccountEmail  string            `json:"serviceAccountEmail,omitempty"`
	AvailableMemory      string            `json:"availableMemory,omitempty"`
	EnvironmentVariables map[string]string `json:"environmentVariables,omitempty"`
}

// GCPCloudRunService represents a GCP Cloud Run service.
type GCPCloudRunService struct {
	Name        string               `json:"name,omitempty"`
	Description string               `json:"description,omitempty"`
	URI         string               `json:"uri,omitempty"`
	Template    *GCPCloudRunTemplate `json:"template,omitempty"`
}

type GCPCloudRunTemplate struct {
	ServiceAccountEmail string                 `json:"serviceAccountEmail,omitempty"`
	Containers          []GCPCloudRunContainer `json:"containers,omitempty"`
}

type GCPCloudRunContainer struct {
	Name  string            `json:"name,omitempty"`
	Image string            `json:"image,omitempty"`
	Env   map[string]string `json:"env,omitempty"`
}

// GCPIAMBinding represents a GCP IAM binding for an AI-related role.
type GCPIAMBinding struct {
	Role    string   `json:"role,omitempty"`
	Members []string `json:"members,omitempty"`
}

// AzureFunctionApp represents an Azure Function App.
type AzureFunctionApp struct {
	Name                     string                    `json:"name,omitempty"`
	ID                       string                    `json:"id,omitempty"`
	Location                 string                    `json:"location,omitempty"`
	ManagedServiceIdentityID string                    `json:"managedServiceIdentityId,omitempty"`
	AppSettings              []AzureFunctionAppSetting `json:"appSettings,omitempty"`
}

type AzureFunctionAppSetting struct {
	Name           string `json:"name,omitempty"`
	HasKeyVaultRef bool   `json:"hasKeyVaultReference,omitempty"`
	IsLikelySecret bool   `json:"isLikelySecret,omitempty"`
}

// LocalAIModel represents a locally cached AI model discovered by the ai.models resource.
type LocalAIModel struct {
	Name          string   `json:"name,omitempty"`
	Source        string   `json:"source,omitempty"`
	Vendor        string   `json:"vendor,omitempty"`
	Family        string   `json:"family,omitempty"`
	Path          string   `json:"path,omitempty"`
	Size          int64    `json:"size,omitempty"`
	ModifiedAt    string   `json:"modifiedAt,omitempty"`
	Format        string   `json:"format,omitempty"`
	Version       string   `json:"version,omitempty"`
	Quantization  string   `json:"quantization,omitempty"`
	ParameterSize string   `json:"parameterSize,omitempty"`
	Architecture  string   `json:"architecture,omitempty"`
	License       string   `json:"license,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Description   string   `json:"description,omitempty"`
}

// SoftwarePackage represents an npm or Python package discovered on the system.
type SoftwarePackage struct {
	Name    string   `json:"name,omitempty"`
	Version string   `json:"version,omitempty"`
	Purl    string   `json:"purl,omitempty"`
	Cpes    []string `json:"cpes,omitempty"`
}

// AiBomFields is the top-level structure for parsing MQL report data.
type AiBomFields struct {
	Asset                   *AiBomAsset                     `json:"asset,omitempty"`
	OllamaModels            []OllamaModel                   `json:"ollama.models,omitempty"`
	HuggingFaceModels       []HuggingFaceModel              `json:"huggingface.models,omitempty"`
	BedrockFoundationModels []AWSBedrockFoundationModel     `json:"aws.bedrock.foundationModels,omitempty"`
	BedrockCustomModels     []AWSBedrockCustomModel         `json:"aws.bedrock.customModels,omitempty"`
	SageMakerModels         []AWSSageMakerModel             `json:"aws.sagemaker.models,omitempty"`
	VertexAIModels          []GCPVertexAIModel              `json:"gcp.project.vertexaiService.models,omitempty"`
	AzureCognitiveAccounts  []AzureCognitiveServicesAccount `json:"azure.subscription.cognitiveServices.accounts,omitempty"`
	VLLM                    *VLLMData                       `json:"vllm,omitempty"`
	ClaudeModels            []ClaudeModel                   `json:"claude.models,omitempty"`
	ClaudeAgents            []ClaudeAgent                   `json:"claude.agents,omitempty"`
	ClaudeSkills            []ClaudeSkill                   `json:"claude.skills,omitempty"`
	ClaudeEnvironments      []ClaudeEnvironment             `json:"claude.environments,omitempty"`
	ClaudeVaults            []ClaudeVault                   `json:"claude.vaults,omitempty"`
	ClaudeMemoryStores      []ClaudeMemoryStore             `json:"claude.memoryStores,omitempty"`
	OpenAIModels            []OpenAIModel                   `json:"openai.models,omitempty"`
	OpenAIFineTuningJobs    []OpenAIFineTuningJob           `json:"openai.fineTuningJobs,omitempty"`
	OpenAIVectorStores      []OpenAIVectorStore             `json:"openai.vectorStores,omitempty"`
	OpenAIProjects          []OpenAIProject                 `json:"openai.projects,omitempty"`
	BedrockGuardrails       []AWSBedrockGuardrail           `json:"aws.bedrock.guardrails,omitempty"`
	BedrockAgents           []AWSBedrockAgent               `json:"aws.bedrock.agents,omitempty"`
	BedrockKnowledgeBases   []AWSBedrockKnowledgeBase       `json:"aws.bedrock.knowledgeBases,omitempty"`
	BedrockFlows            []AWSBedrockFlow                `json:"aws.bedrock.flows,omitempty"`
	VertexAIPipelineJobs    []GCPVertexAIPipelineJob        `json:"gcp.project.vertexaiService.pipelineJobs,omitempty"`
	VertexAIDatasets        []GCPVertexAIDataset            `json:"gcp.project.vertexaiService.datasets,omitempty"`
	ModelArmorTemplates     []GCPModelArmorTemplate         `json:"gcp.project.modelArmorService.templates,omitempty"`
	ModelArmorFloorSetting  *GCPModelArmorFloorSetting      `json:"gcp.project.modelArmorService.floorSetting,omitempty"`
	AzureRAIAccounts        []AzureRAIAccount               `json:"azure.subscription.cognitiveServices.accounts.rai,omitempty"`
	ClaudeCode              *CodingAgent                    `json:"claude.code,omitempty"`
	OpenAICodex             *CodingAgent                    `json:"openai.codex,omitempty"`
	Cursor                  *CodingAgent                    `json:"cursor,omitempty"`
	GithubCopilot           *CodingAgent                    `json:"github.copilot,omitempty"`
	Windsurf                *CodingAgent                    `json:"windsurf,omitempty"`
	Gemini                  *CodingAgent                    `json:"gemini,omitempty"`
	Goose                   *CodingAgent                    `json:"goose,omitempty"`
	Zed                     *CodingAgent                    `json:"zed,omitempty"`
	Roo                     *CodingAgent                    `json:"roo,omitempty"`
	Cline                   *CodingAgent                    `json:"cline,omitempty"`
	Kiro                    *CodingAgent                    `json:"kiro,omitempty"`
	Trae                    *CodingAgent                    `json:"trae,omitempty"`
	Junie                   *CodingAgent                    `json:"junie,omitempty"`
	Augment                 *CodingAgent                    `json:"augment,omitempty"`
	Kilocode                *CodingAgent                    `json:"kilocode,omitempty"`
	Continuedev             *CodingAgent                    `json:"continuedev,omitempty"`
	MistralVibe             *CodingAgent                    `json:"mistral.vibe,omitempty"`
	Antigravity             *CodingAgent                    `json:"antigravity,omitempty"`
	IbmBob                  *CodingAgent                    `json:"ibm.bob,omitempty"`
	OpenClaw                *CodingAgent                    `json:"openclaw,omitempty"`
	SnowflakeCortex         *CodingAgent                    `json:"snowflake.cortex,omitempty"`
	Warp                    *CodingAgent                    `json:"warp,omitempty"`
	OpenHands               *CodingAgent                    `json:"openhands,omitempty"`
	OpenCode                *CodingAgent                    `json:"opencode,omitempty"`
	Pi                      *CodingAgent                    `json:"pi,omitempty"`
	QwenCode                *CodingAgent                    `json:"qwen.code,omitempty"`
	LambdaFunctions         []AWSLambdaFunction             `json:"aws.lambda.functions,omitempty"`
	GCPCloudFunctions       []GCPCloudFunction              `json:"gcp.project.cloudFunctions,omitempty"`
	GCPCloudRunServices     []GCPCloudRunService            `json:"gcp.project.cloudRunService.services,omitempty"`
	GCPIAMBindings          []GCPIAMBinding                 `json:"gcp.project.iamPolicy,omitempty"`
	AzureFunctionApps       []AzureFunctionApp              `json:"azure.subscription.functionsService.functions,omitempty"`
	NpmPackages             []SoftwarePackage               `json:"npm.packages,omitempty"`
	PythonPackages          []SoftwarePackage               `json:"python.packages,omitempty"`
	LocalModels             []LocalAIModel                  `json:"ai.models,omitempty"`
}
