// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generator

import (
	"encoding/json"
	"fmt"
	"maps"
	"sort"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnspec/v13"
	"go.mondoo.com/cnspec/v13/internal/aibom"
	cr "go.mondoo.com/mql/v13/cli/reporter"
	"go.mondoo.com/mql/v13/utils/sortx"
)

// GenerateAiBom generates an AI Bill of Materials from a cnspec report.
func GenerateAiBom(r *cr.Report) []*aibom.AiBom {
	if r == nil {
		return nil
	}

	gen := &aibom.Generator{
		Vendor:  "Mondoo, Inc.",
		Name:    "cnspec",
		Version: cnspec.Version,
		Url:     "https://mondoo.com",
	}
	now := time.Now().UTC().Format(time.RFC3339)

	boms := []*aibom.AiBom{}
	for assetMrn, asset := range r.Assets {
		bom := &aibom.AiBom{
			Generator: gen,
			Timestamp: now,
			Status:    aibom.Status_STATUS_SUCCEEDED,
			Asset: &aibom.Asset{
				Name: asset.GetName(),
			},
		}

		dataPoints := r.Data[assetMrn]
		if dataPoints == nil {
			bom.Status = aibom.Status_STATUS_FAILED
			bom.Errors = append(bom.Errors, "no data points found")
			boms = append(boms, bom)
			continue
		}

		// Collected sub-collections for downstream helpers (parsed once)
		var allNpm, allPython []SoftwarePackage
		var allLambdas []AWSLambdaFunction
		var allGCPFunctions []GCPCloudFunction
		var allGCPRunServices []GCPCloudRunService
		var allAzureFuncApps []AzureFunctionApp
		var allGCPIAMBindings []GCPIAMBinding

		keys := sortx.Keys(dataPoints.Values)
		for _, k := range keys {
			dataValue := dataPoints.Values[k]
			jsondata, err := cr.JsonValue(dataValue.Content)
			if err != nil {
				bom.Status = aibom.Status_STATUS_PARTIALLY_SUCCEEDED
				bom.Errors = append(bom.Errors, errors.Wrap(err, "failed to parse json data").Error())
				continue
			}

			fields := AiBomFields{}
			err = json.Unmarshal(jsondata, &fields)
			if err != nil {
				bom.Status = aibom.Status_STATUS_PARTIALLY_SUCCEEDED
				bom.Errors = append(bom.Errors, errors.Wrap(err, "failed to parse aibom fields").Error())
				continue
			}

			// Collect sub-collections for downstream processing
			allNpm = append(allNpm, fields.NpmPackages...)
			allPython = append(allPython, fields.PythonPackages...)
			allLambdas = append(allLambdas, fields.LambdaFunctions...)
			allGCPFunctions = append(allGCPFunctions, fields.GCPCloudFunctions...)
			allGCPRunServices = append(allGCPRunServices, fields.GCPCloudRunServices...)
			allAzureFuncApps = append(allAzureFuncApps, fields.AzureFunctionApps...)
			allGCPIAMBindings = append(allGCPIAMBindings, fields.GCPIAMBindings...)

			if fields.Asset != nil {
				bom.Asset.Name = fields.Asset.Name
				bom.Asset.PlatformIds = fields.Asset.IDs
				bom.Asset.Platform = &aibom.Platform{
					Name:    fields.Asset.Platform,
					Version: fields.Asset.Version,
					Arch:    fields.Asset.Arch,
					Family:  fields.Asset.Family,
				}
			}

			// Process locally cached AI models (filesystem detection)
			for _, m := range fields.LocalModels {
				bom.Models = append(bom.Models, localModelToComponent(m))
			}

			// Process Ollama models
			for _, m := range fields.OllamaModels {
				bom.Models = append(bom.Models, ollamaToModelComponent(m))
			}

			// Process HuggingFace models
			for _, m := range fields.HuggingFaceModels {
				bom.Models = append(bom.Models, huggingfaceToModelComponent(m))
			}

			// Process AWS Bedrock foundation models
			for _, m := range fields.BedrockFoundationModels {
				bom.Models = append(bom.Models, bedrockFoundationToModelComponent(m))
			}

			// Process AWS Bedrock custom models
			for _, m := range fields.BedrockCustomModels {
				bom.Models = append(bom.Models, bedrockCustomToModelComponent(m))
			}

			// Process AWS SageMaker models
			for _, m := range fields.SageMakerModels {
				bom.Models = append(bom.Models, sagemakerToModelComponent(m))
			}

			// Process GCP Vertex AI models
			for _, m := range fields.VertexAIModels {
				bom.Models = append(bom.Models, vertexaiToModelComponent(m))
			}

			// Process Azure Cognitive Services accounts
			for _, m := range fields.AzureCognitiveAccounts {
				if m.Kind == "OpenAI" {
					bom.Models = append(bom.Models, azureOpenAIToModelComponent(m))
				}
			}

			// Process vLLM models and server
			if fields.VLLM != nil {
				if len(fields.VLLM.Models) > 0 {
					for _, m := range fields.VLLM.Models {
						bom.Models = append(bom.Models, vllmModelToComponent(m, fields.VLLM.Server))
					}
				} else {
					bom.Models = append(bom.Models, vllmToModelComponent(fields.VLLM))
				}
			}

			// Process Claude API models
			for _, m := range fields.ClaudeModels {
				bom.Models = append(bom.Models, claudeModelToComponent(m))
			}

			// Process Claude managed agents (skills are org-level, attach to first agent only)
			for i, a := range fields.ClaudeAgents {
				var skills []ClaudeSkill
				if i == 0 {
					skills = fields.ClaudeSkills
				}
				bom.Agents = append(bom.Agents, claudeAgentToComponent(a, skills))
			}

			// Process Claude memory stores as knowledge bases
			for _, ms := range fields.ClaudeMemoryStores {
				bom.KnowledgeBases = append(bom.KnowledgeBases, claudeMemoryStoreToKB(ms))
			}

			// Process OpenAI API models
			for _, m := range fields.OpenAIModels {
				bom.Models = append(bom.Models, openaiModelToComponent(m, fields.OpenAIFineTuningJobs))
			}

			// Process OpenAI vector stores as knowledge bases
			for _, vs := range fields.OpenAIVectorStores {
				bom.KnowledgeBases = append(bom.KnowledgeBases, openaiVectorStoreToKB(vs))
			}

			// Process AWS Bedrock guardrails
			for _, g := range fields.BedrockGuardrails {
				bom.Guardrails = append(bom.Guardrails, bedrockGuardrailToComponent(g))
			}

			// Process AWS Bedrock agents
			for _, a := range fields.BedrockAgents {
				bom.Agents = append(bom.Agents, bedrockAgentToComponent(a))
			}

			// Process AWS Bedrock knowledge bases
			for _, kb := range fields.BedrockKnowledgeBases {
				bom.KnowledgeBases = append(bom.KnowledgeBases, bedrockKBToComponent(kb))
			}

			// Process AWS Bedrock flows as agents (workflow orchestration)
			for _, f := range fields.BedrockFlows {
				bom.Agents = append(bom.Agents, bedrockFlowToComponent(f))
			}

			// Process GCP Model Armor as guardrails
			for _, t := range fields.ModelArmorTemplates {
				bom.Guardrails = append(bom.Guardrails, modelArmorToGuardrail(t))
			}
			if fields.ModelArmorFloorSetting != nil && fields.ModelArmorFloorSetting.Name != "" {
				bom.Guardrails = append(bom.Guardrails, modelArmorFloorToGuardrail(fields.ModelArmorFloorSetting))
			}

			// Process GCP Cloud Functions as scannable dependencies
			for _, fn := range fields.GCPCloudFunctions {
				bom.Agents = append(bom.Agents, gcpCloudFuncToComponent(fn))
			}

			// Process local coding agents
			agentMap := map[string]*CodingAgent{
				"claude.code":      fields.ClaudeCode,
				"openai.codex":     fields.OpenAICodex,
				"cursor":           fields.Cursor,
				"github.copilot":   fields.GithubCopilot,
				"windsurf":         fields.Windsurf,
				"gemini":           fields.Gemini,
				"goose":            fields.Goose,
				"zed":              fields.Zed,
				"roo":              fields.Roo,
				"cline":            fields.Cline,
				"kiro":             fields.Kiro,
				"trae":             fields.Trae,
				"junie":            fields.Junie,
				"augment":          fields.Augment,
				"kilocode":         fields.Kilocode,
				"continuedev":      fields.Continuedev,
				"mistral.vibe":     fields.MistralVibe,
				"antigravity":      fields.Antigravity,
				"ibm.bob":          fields.IbmBob,
				"openclaw":         fields.OpenClaw,
				"snowflake.cortex": fields.SnowflakeCortex,
				"warp":             fields.Warp,
				"openhands":        fields.OpenHands,
				"opencode":         fields.OpenCode,
				"pi":               fields.Pi,
				"qwen.code":        fields.QwenCode,
			}
			for _, name := range sortx.Keys(agentMap) {
				agent := agentMap[name]
				if agent != nil && agent.ConfigPath != "" {
					bom.Agents = append(bom.Agents, codingAgentToComponent(name, agent))
				}
			}
		}

		// Classify AI dependencies from discovered software packages
		bom.AIDependencies = classifyAIDependencies(allNpm, allPython)

		// Detect compute services with AI access
		detectComputeAIAccess(bom, allLambdas, allGCPFunctions, allGCPRunServices, allAzureFuncApps, allGCPIAMBindings)

		// Correlate MCP server packages with discovered npm/python packages
		correlateAgentPackages(bom.Agents, allNpm, allPython)

		// Enrich Bedrock agent dependencies with Lambda function metadata
		enrichAgentLambdas(bom.Agents, allLambdas)

		bom.Completeness = ComputeCompleteness(bom.Models)
		boms = append(boms, bom)
	}
	return boms
}

func correlateAgentPackages(agents []*aibom.AgentComponent, npmPkgs, pythonPkgs []SoftwarePackage) {
	npmIdx := map[string]*SoftwarePackage{}
	pyIdx := map[string]*SoftwarePackage{}
	for i := range npmPkgs {
		npmIdx[npmPkgs[i].Name] = &npmPkgs[i]
	}
	for i := range pythonPkgs {
		pyIdx[pythonPkgs[i].Name] = &pythonPkgs[i]
	}

	if len(npmIdx) == 0 && len(pyIdx) == 0 {
		return
	}

	for _, agent := range agents {
		for _, mcp := range agent.McpServers {
			pkgName := mcpPackageName(mcp)
			if pkgName == "" {
				continue
			}

			if p, ok := npmIdx[pkgName]; ok {
				mcp.Packages = append(mcp.Packages, &aibom.PackageRef{
					Name:    p.Name,
					Version: p.Version,
					Purl:    p.Purl,
					Cpes:    p.Cpes,
				})
			}
			if p, ok := pyIdx[pkgName]; ok {
				mcp.Packages = append(mcp.Packages, &aibom.PackageRef{
					Name:    p.Name,
					Version: p.Version,
					Purl:    p.Purl,
					Cpes:    p.Cpes,
				})
			}
		}
	}
}

// aiPackageRegistry maps known AI/ML package names to their category.
var aiPackageRegistry = map[string]string{
	// Python - Model frameworks
	"transformers":          "model-framework",
	"torch":                 "model-framework",
	"torchvision":           "model-framework",
	"torchaudio":            "model-framework",
	"tensorflow":            "model-framework",
	"tensorflow-gpu":        "model-framework",
	"keras":                 "model-framework",
	"onnxruntime":           "model-framework",
	"onnxruntime-gpu":       "model-framework",
	"diffusers":             "model-framework",
	"sentence-transformers": "model-framework",
	"accelerate":            "model-framework",
	"safetensors":           "model-framework",
	"optimum":               "model-framework",
	"peft":                  "model-framework",
	"trl":                   "model-framework",
	"jax":                   "model-framework",
	"flax":                  "model-framework",
	// Python - API clients
	"openai":              "api-client",
	"anthropic":           "api-client",
	"google-generativeai": "api-client",
	"cohere":              "api-client",
	"mistralai":           "api-client",
	"replicate":           "api-client",
	"together":            "api-client",
	"groq":                "api-client",
	"fireworks-ai":        "api-client",
	"voyageai":            "api-client",
	// Python - Agent frameworks
	"langchain":           "agent-framework",
	"langchain-core":      "agent-framework",
	"langchain-community": "agent-framework",
	"langchain-openai":    "agent-framework",
	"langchain-anthropic": "agent-framework",
	"langgraph":           "agent-framework",
	"crewai":              "agent-framework",
	"autogen":             "agent-framework",
	"pyautogen":           "agent-framework",
	"phidata":             "agent-framework",
	"smolagents":          "agent-framework",
	"llama-index":         "agent-framework",
	"llama-index-core":    "agent-framework",
	"dspy-ai":             "agent-framework",
	"semantic-kernel":     "agent-framework",
	"haystack-ai":         "agent-framework",
	// Python - Vector databases
	"chromadb":        "vector-db",
	"pinecone-client": "vector-db",
	"weaviate-client": "vector-db",
	"qdrant-client":   "vector-db",
	"pymilvus":        "vector-db",
	"pgvector":        "vector-db",
	"lancedb":         "vector-db",
	// Python - ML tools
	"mlflow":          "ml-tool",
	"wandb":           "ml-tool",
	"huggingface-hub": "ml-tool",
	"datasets":        "ml-tool",
	"vllm":            "ml-tool",
	"bentoml":         "ml-tool",
	"ray":             "ml-tool",
	"unsloth":         "ml-tool",
	// npm - AI SDKs (langchain, openai, chromadb shared with Python above)
	"@anthropic-ai/sdk":           "api-client",
	"@google/generative-ai":       "api-client",
	"@mistralai/mistralai":        "api-client",
	"@langchain/core":             "agent-framework",
	"@langchain/openai":           "agent-framework",
	"@langchain/anthropic":        "agent-framework",
	"@langchain/community":        "agent-framework",
	"ai":                          "api-client",
	"@ai-sdk/openai":              "api-client",
	"@ai-sdk/anthropic":           "api-client",
	"@ai-sdk/google":              "api-client",
	"llamaindex":                  "agent-framework",
	"@pinecone-database/pinecone": "vector-db",
	"@qdrant/js-client-rest":      "vector-db",
}

func classifyAIDependencies(npmPkgs, pythonPkgs []SoftwarePackage) []*aibom.AIDependency {
	var deps []*aibom.AIDependency
	seen := map[string]bool{}

	for _, pkg := range pythonPkgs {
		cat, ok := aiPackageRegistry[strings.ToLower(pkg.Name)]
		if !ok {
			continue
		}
		key := "python:" + strings.ToLower(pkg.Name)
		if seen[key] {
			continue
		}
		seen[key] = true
		deps = append(deps, &aibom.AIDependency{
			Name:     pkg.Name,
			Version:  pkg.Version,
			Category: cat,
			Purl:     pkg.Purl,
			Language: "python",
		})
	}

	for _, pkg := range npmPkgs {
		cat, ok := aiPackageRegistry[strings.ToLower(pkg.Name)]
		if !ok {
			continue
		}
		key := "npm:" + strings.ToLower(pkg.Name)
		if seen[key] {
			continue
		}
		seen[key] = true
		deps = append(deps, &aibom.AIDependency{
			Name:     pkg.Name,
			Version:  pkg.Version,
			Category: cat,
			Purl:     pkg.Purl,
			Language: "javascript",
		})
	}

	return deps
}

// AI service action prefixes to detect in IAM policies.
var aiActionPrefixes = []string{
	"bedrock:", "sagemaker:", "comprehend:", "rekognition:",
	"textract:", "translate:", "polly:", "transcribe:", "lex:",
}

// Environment variable keys that hint at AI service usage.
var aiEnvKeywords = []string{
	"BEDROCK", "SAGEMAKER", "OPENAI", "ANTHROPIC", "VERTEX",
	"AI_MODEL", "MODEL_ID", "LLM", "GEMINI", "CLAUDE",
	"HUGGINGFACE", "INFERENCE", "EMBEDDING",
}

func detectComputeAIAccess(bom *aibom.AiBom, lambdas []AWSLambdaFunction, gcpFuncs []GCPCloudFunction, gcpRun []GCPCloudRunService, azureFuncs []AzureFunctionApp, gcpBindings []GCPIAMBinding) {
	// Build GCP IAM AI-role index: service account email → roles
	gcpAIRoles := map[string][]string{}
	for _, binding := range gcpBindings {
		for _, member := range binding.Members {
			gcpAIRoles[member] = append(gcpAIRoles[member], binding.Role)
		}
	}

	for _, fn := range lambdas {
		ca := detectLambdaAIAccess(fn)
		if ca != nil {
			bom.ComputeAccess = append(bom.ComputeAccess, ca)
		}
	}
	for _, fn := range gcpFuncs {
		ca := detectCloudFuncAIAccess(fn, gcpAIRoles)
		if ca != nil {
			bom.ComputeAccess = append(bom.ComputeAccess, ca)
		}
	}
	for _, svc := range gcpRun {
		ca := detectCloudRunAIAccess(svc, gcpAIRoles)
		if ca != nil {
			bom.ComputeAccess = append(bom.ComputeAccess, ca)
		}
	}
	for _, app := range azureFuncs {
		ca := detectAzureFuncAIAccess(app)
		if ca != nil {
			bom.ComputeAccess = append(bom.ComputeAccess, ca)
		}
	}
}

func detectLambdaAIAccess(fn AWSLambdaFunction) *aibom.ComputeAIAccess {
	aiActions := extractAIActionsFromRole(fn.Role)
	envHints := extractAIEnvHints(fn.Environment)

	if len(aiActions) == 0 && len(envHints) == 0 {
		return nil
	}

	services := map[string]bool{}
	for _, action := range aiActions {
		svc := strings.SplitN(action, ":", 2)[0]
		services[svc] = true
	}

	ca := &aibom.ComputeAIAccess{
		Name:      fn.Name,
		Type:      "lambda",
		Provider:  "aws",
		Arn:       fn.Arn,
		Runtime:   fn.Runtime,
		CodeHash:  fn.CodeSha256,
		AIActions: aiActions,
		EnvHints:  envHints,
		Labels: map[string]string{
			"handler":      fn.Handler,
			"package_type": fn.PackageType,
		},
	}
	if fn.ImageUri != "" {
		ca.ImageUri = fn.ImageUri
	} else if fn.ResolvedImageUri != "" {
		ca.ImageUri = fn.ResolvedImageUri
	}
	if fn.Role != nil {
		ca.ServiceAccount = fn.Role.Arn
	}
	for svc := range services {
		ca.AIServices = append(ca.AIServices, svc)
	}
	sort.Strings(ca.AIServices)
	return ca
}

func detectCloudFuncAIAccess(fn GCPCloudFunction, aiRoles map[string][]string) *aibom.ComputeAIAccess {
	sa := ""
	if fn.ServiceConfig != nil {
		sa = fn.ServiceConfig.ServiceAccountEmail
	}

	// Check if the service account has AI-related IAM roles
	saKey := "serviceAccount:" + sa
	roles := aiRoles[saKey]

	// Check environment variables for AI hints
	envHints := map[string]string{}
	if fn.ServiceConfig != nil {
		envHints = extractAIEnvHints(fn.ServiceConfig.EnvironmentVariables)
	}

	if len(roles) == 0 && len(envHints) == 0 {
		return nil
	}

	ca := &aibom.ComputeAIAccess{
		Name:           fn.Name,
		Type:           "cloud-function",
		Provider:       "gcp",
		ServiceAccount: sa,
		AIServices:     roles,
		EnvHints:       envHints,
		Labels: map[string]string{
			"state":       fn.State,
			"environment": fn.Environment,
		},
	}
	if fn.BuildConfig != nil {
		ca.Runtime = fn.BuildConfig.Runtime
		if fn.BuildConfig.DockerRepository != "" {
			ca.ImageUri = fn.BuildConfig.DockerRepository
		}
	}
	if fn.URL != "" {
		ca.Labels["url"] = fn.URL
	}
	return ca
}

func detectCloudRunAIAccess(svc GCPCloudRunService, aiRoles map[string][]string) *aibom.ComputeAIAccess {
	sa := ""
	if svc.Template != nil {
		sa = svc.Template.ServiceAccountEmail
	}

	saKey := "serviceAccount:" + sa
	roles := aiRoles[saKey]

	// Check container env vars for AI hints
	envHints := map[string]string{}
	if svc.Template != nil {
		for _, c := range svc.Template.Containers {
			maps.Copy(envHints, extractAIEnvHints(c.Env))
		}
	}

	if len(roles) == 0 && len(envHints) == 0 {
		return nil
	}

	ca := &aibom.ComputeAIAccess{
		Name:           svc.Name,
		Type:           "cloud-run",
		Provider:       "gcp",
		ServiceAccount: sa,
		AIServices:     roles,
		EnvHints:       envHints,
		Labels:         map[string]string{},
	}
	if svc.URI != "" {
		ca.Labels["url"] = svc.URI
	}
	if svc.Template != nil {
		for _, c := range svc.Template.Containers {
			if c.Image != "" {
				ca.ImageUri = c.Image
				break
			}
		}
	}
	return ca
}

func detectAzureFuncAIAccess(app AzureFunctionApp) *aibom.ComputeAIAccess {
	envHints := map[string]string{}
	for _, s := range app.AppSettings {
		for _, kw := range aiEnvKeywords {
			if strings.Contains(strings.ToUpper(s.Name), kw) {
				envHints[s.Name] = "(configured)"
				break
			}
		}
	}

	if len(envHints) == 0 && app.ManagedServiceIdentityID == "" {
		return nil
	}

	ca := &aibom.ComputeAIAccess{
		Name:     app.Name,
		Type:     "function-app",
		Provider: "azure",
		Arn:      app.ID,
		EnvHints: envHints,
		Labels: map[string]string{
			"location": app.Location,
		},
	}
	if app.ManagedServiceIdentityID != "" {
		ca.ServiceAccount = app.ManagedServiceIdentityID
	}
	return ca
}

func extractAIActionsFromRole(role *AWSIAMRole) []string {
	if role == nil {
		return nil
	}
	var aiActions []string
	for _, policy := range role.AttachedPolicies {
		if policy.DefaultVersion == nil || policy.DefaultVersion.Document == nil {
			continue
		}
		actions := extractActionsFromPolicyDoc(policy.DefaultVersion.Document)
		for _, action := range actions {
			for _, prefix := range aiActionPrefixes {
				if strings.HasPrefix(strings.ToLower(action), prefix) {
					aiActions = append(aiActions, action)
					break
				}
			}
		}
	}
	return deduplicate(aiActions)
}

func extractActionsFromPolicyDoc(doc map[string]any) []string {
	var actions []string
	stmts, ok := doc["Statement"]
	if !ok {
		return nil
	}
	stmtList, ok := stmts.([]any)
	if !ok {
		return nil
	}
	for _, s := range stmtList {
		stmt, ok := s.(map[string]any)
		if !ok {
			continue
		}
		effect, _ := stmt["Effect"].(string)
		if strings.ToLower(effect) != "allow" {
			continue
		}
		switch a := stmt["Action"].(type) {
		case string:
			actions = append(actions, a)
		case []any:
			for _, v := range a {
				if s, ok := v.(string); ok {
					actions = append(actions, s)
				}
			}
		}
	}
	return actions
}

func extractAIEnvHints(env map[string]string) map[string]string {
	if len(env) == 0 {
		return nil
	}
	hints := map[string]string{}
	for k := range env {
		for _, kw := range aiEnvKeywords {
			if strings.Contains(strings.ToUpper(k), kw) {
				hints[k] = "(configured)"
				break
			}
		}
	}
	if len(hints) == 0 {
		return nil
	}
	return hints
}

func enrichAgentLambdas(agents []*aibom.AgentComponent, lambdas []AWSLambdaFunction) {
	lambdaIdx := map[string]*AWSLambdaFunction{}
	for i := range lambdas {
		lambdaIdx[lambdas[i].Arn] = &lambdas[i]
	}

	if len(lambdaIdx) == 0 {
		return
	}

	for _, agent := range agents {
		if agent.Provider != "aws-bedrock" {
			continue
		}
		for _, dep := range agent.Dependencies {
			if dep.Type != "action-group" || dep.Arn == "" {
				continue
			}
			fn, ok := lambdaIdx[dep.Arn]
			if !ok {
				continue
			}
			dep.Runtime = fn.Runtime
			dep.CodeHash = fn.CodeSha256
			if fn.ImageUri != "" {
				dep.ImageUri = fn.ImageUri
			} else if fn.ResolvedImageUri != "" {
				dep.ImageUri = fn.ResolvedImageUri
			}
			for _, layer := range fn.Layers {
				dep.Layers = append(dep.Layers, layer.Arn)
			}
			dep.Labels = map[string]string{
				"handler":      fn.Handler,
				"package_type": fn.PackageType,
				"code_size":    fmt.Sprintf("%d", fn.CodeSize),
			}
		}
	}
}

func mcpPackageName(mcp *aibom.McpServer) string {
	if mcp.Command == "" {
		return ""
	}
	args := mcp.Args
	switch mcp.Command {
	case "npx", "bunx":
		if len(args) > 0 {
			pkg := args[0]
			if pkg == "-y" && len(args) > 1 {
				pkg = args[1]
			}
			return pkg
		}
	case "uvx", "pipx":
		if len(args) > 0 {
			return args[0]
		}
	case "node":
		if len(args) > 0 {
			return args[0]
		}
	case "python", "python3":
		for i, a := range args {
			if a == "-m" && i+1 < len(args) {
				return args[i+1]
			}
		}
	}
	return ""
}

func localModelToComponent(m LocalAIModel) *aibom.ModelComponent {
	mc := &aibom.ModelComponent{
		Name:               m.Name,
		Provider:           m.Source,
		Author:             m.Vendor,
		ArchitectureFamily: m.Family,
		ModelArchitecture:  m.Architecture,
		Format:             m.Format,
		Version:            m.Version,
		Quantization:       m.Quantization,
		ParameterSize:      m.ParameterSize,
		License:            m.License,
		Description:        m.Description,
		Tags:               m.Tags,
		Labels:             map[string]string{},
		Provenance: map[string]string{
			"detection_source": "ai.models",
		},
	}

	if m.Path != "" {
		mc.SourceUrl = "file://" + m.Path
		mc.Provenance["local_path"] = m.Path
	}
	if m.Size > 0 {
		mc.Labels["size_bytes"] = fmt.Sprintf("%d", m.Size)
	}
	if m.ModifiedAt != "" {
		mc.UpdatedAt = m.ModifiedAt
	}

	mc.Purl = localModelPurl(m)

	return mc
}

func localModelPurl(m LocalAIModel) string {
	name := m.Name
	vendor := m.Vendor
	version := m.Version

	switch m.Source {
	case "ollama":
		if vendor != "" {
			return fmt.Sprintf("pkg:ollama/%s/%s@%s", vendor, name, version)
		}
		return fmt.Sprintf("pkg:ollama/%s@%s", name, version)
	case "huggingface":
		return fmt.Sprintf("pkg:huggingface/%s/%s@%s", vendor, name, version)
	default:
		source := m.Source
		if source == "" {
			source = "generic"
		}
		if vendor != "" {
			return fmt.Sprintf("pkg:generic/%s/%s/%s@%s", source, vendor, name, version)
		}
		return fmt.Sprintf("pkg:generic/%s/%s@%s", source, name, version)
	}
}

func ollamaToModelComponent(m OllamaModel) *aibom.ModelComponent {
	mc := &aibom.ModelComponent{
		Name:               m.Name,
		Provider:           "ollama",
		ModelId:            m.Model,
		ArchitectureFamily: m.Family,
		Format:             m.Format,
		ParameterSize:      m.ParameterSize,
		Quantization:       m.QuantizationLevel,
		Capabilities:       m.Capabilities,
		UpdatedAt:          m.ModifiedAt,
		Labels:             map[string]string{},
		Provenance:         map[string]string{},
	}

	// Extract version from name (after colon)
	if parts := strings.SplitN(m.Name, ":", 2); len(parts) == 2 {
		mc.Version = parts[1]
	}

	// Use digest as version fallback
	if mc.Version == "" && m.Digest != "" {
		mc.Version = m.Digest[:12]
	}

	// Enrich from info block
	if m.Info.Architecture != "" {
		mc.ModelArchitecture = m.Info.Architecture
	}
	if m.Info.Author != "" {
		mc.Author = m.Info.Author
	}
	if m.Info.Description != "" {
		mc.Description = m.Info.Description
	}
	if m.Info.License != "" {
		mc.License = m.Info.License
	} else if m.License != "" {
		mc.License = m.License
	}
	if len(m.Info.Datasets) > 0 {
		mc.TrainingDatasets = m.Info.Datasets
	}
	if len(m.Info.Tags) > 0 {
		mc.Tags = m.Info.Tags
	}
	if len(m.Info.Languages) > 0 {
		mc.Labels["languages"] = strings.Join(m.Info.Languages, ",")
	}
	if m.Info.ParameterCount > 0 {
		mc.Provenance["parameter_count"] = fmt.Sprintf("%d", m.Info.ParameterCount)
	}
	if m.Info.ContextLength > 0 {
		mc.Provenance["context_length"] = fmt.Sprintf("%d", m.Info.ContextLength)
	}

	// Generate PURL
	vendor := m.Info.Author
	name := m.Name
	if parts := strings.SplitN(m.Name, ":", 2); len(parts) == 2 {
		name = parts[0]
	}
	if vendor != "" {
		mc.Purl = fmt.Sprintf("pkg:ollama/%s/%s@%s", vendor, name, mc.Version)
	} else {
		mc.Purl = fmt.Sprintf("pkg:ollama/%s@%s", name, mc.Version)
	}

	return mc
}

func huggingfaceToModelComponent(m HuggingFaceModel) *aibom.ModelComponent {
	// Split ID into group (author) and short name
	shortName := m.ID
	author := m.Author
	if parts := strings.SplitN(m.ID, "/", 2); len(parts) == 2 {
		if author == "" {
			author = parts[0]
		}
		shortName = parts[1]
	}

	mc := &aibom.ModelComponent{
		Name:     shortName,
		Provider: "huggingface",
		ModelId:  m.ModelID,
		Author:   author,
		License:  m.License,
		Task:     m.PipelineTag,
		Tags:     m.Tags,
		Labels:   map[string]string{},
		Provenance: map[string]string{
			"library":   m.LibraryName,
			"full_name": m.ID,
		},
	}

	if m.SHA != "" && len(m.SHA) >= 8 {
		mc.Version = m.SHA[:8]
		mc.Provenance["sha"] = m.SHA
	}
	if m.CreatedAt != "" {
		mc.CreatedAt = m.CreatedAt
	}
	if m.LastModified != "" {
		mc.UpdatedAt = m.LastModified
	}

	// Extract architecture from config
	if m.Config != nil {
		if archs, ok := m.Config["architectures"]; ok {
			if archList, ok := archs.([]any); ok && len(archList) > 0 {
				mc.ModelArchitecture = fmt.Sprintf("%v", archList[0])
			}
		}
		if modelType, ok := m.Config["model_type"]; ok {
			mc.ArchitectureFamily = fmt.Sprintf("%v", modelType)
		}
		if vocabSize, ok := m.Config["vocab_size"]; ok {
			mc.Labels["vocab_size"] = fmt.Sprintf("%v", vocabSize)
		}
		if tokenizerClass, ok := m.Config["tokenizer_class"]; ok {
			mc.Labels["tokenizer_class"] = fmt.Sprintf("%v", tokenizerClass)
		}
	}

	// Derive input/output modalities from pipeline tag
	mc.InputModalities, mc.OutputModalities = pipelineTagToModalities(m.PipelineTag)

	// Extract metadata from cardData
	if m.CardData != nil {
		if datasets, ok := m.CardData["datasets"]; ok {
			if dsList, ok := datasets.([]any); ok {
				for _, ds := range dsList {
					mc.TrainingDatasets = append(mc.TrainingDatasets, fmt.Sprintf("%v", ds))
				}
			}
		}
		// Extract datasets from model-index (used by many HuggingFace models)
		if len(mc.TrainingDatasets) == 0 {
			if modelIndex, ok := m.CardData["model-index"]; ok {
				if miList, ok := modelIndex.([]any); ok {
					seen := map[string]bool{}
					for _, mi := range miList {
						miMap, ok := mi.(map[string]any)
						if !ok {
							continue
						}
						results, ok := miMap["results"].([]any)
						if !ok {
							continue
						}
						for _, result := range results {
							rMap, ok := result.(map[string]any)
							if !ok {
								continue
							}
							ds, ok := rMap["dataset"].(map[string]any)
							if !ok {
								continue
							}
							dsType := fmt.Sprintf("%v", ds["type"])
							if dsType != "" && !seen[dsType] {
								mc.TrainingDatasets = append(mc.TrainingDatasets, dsType)
								seen[dsType] = true
							}
						}
					}
				}
			}
		}
		if baseModel, ok := m.CardData["base_model"]; ok {
			mc.Provenance["base_model"] = fmt.Sprintf("%v", baseModel)
		}
	}

	// Extract arxiv paper references from tags
	for _, tag := range m.Tags {
		if arxivID, ok := strings.CutPrefix(tag, "arxiv:"); ok {
			mc.Provenance["arxiv"] = arxivID
		}
	}

	// Generate PURL with namespace
	if mc.Author != "" {
		mc.Purl = fmt.Sprintf("pkg:huggingface/%s/%s@%s", mc.Author, shortName, mc.Version)
	} else {
		mc.Purl = fmt.Sprintf("pkg:huggingface/%s@%s", m.ID, mc.Version)
	}
	mc.SourceUrl = fmt.Sprintf("https://huggingface.co/%s", m.ID)

	return mc
}

func pipelineTagToModalities(tag string) (inputs []string, outputs []string) {
	switch tag {
	case "automatic-speech-recognition":
		return []string{"audio"}, []string{"text"}
	case "text-to-speech":
		return []string{"text"}, []string{"audio"}
	case "text-generation", "text2text-generation", "translation",
		"summarization", "fill-mask", "question-answering", "text-classification",
		"token-classification", "table-question-answering", "conversational":
		return []string{"text"}, []string{"text"}
	case "image-classification", "image-to-text", "visual-question-answering":
		return []string{"image"}, []string{"text"}
	case "text-to-image":
		return []string{"text"}, []string{"image"}
	case "image-to-image":
		return []string{"image"}, []string{"image"}
	case "text-to-video", "image-to-video":
		return []string{"text"}, []string{"video"}
	case "feature-extraction", "sentence-similarity":
		return []string{"text"}, []string{"tensor"}
	case "object-detection", "image-segmentation":
		return []string{"image"}, []string{"tensor"}
	case "audio-classification":
		return []string{"audio"}, []string{"text"}
	case "video-classification":
		return []string{"video"}, []string{"text"}
	default:
		return nil, nil
	}
}

func bedrockFoundationToModelComponent(m AWSBedrockFoundationModel) *aibom.ModelComponent {
	mc := &aibom.ModelComponent{
		Name:             m.ModelName,
		Provider:         "aws-bedrock",
		ModelId:          m.ModelID,
		Author:           m.ProviderName,
		InputModalities:  m.InputModalities,
		OutputModalities: m.OutputModalities,
		Purl:             fmt.Sprintf("pkg:aws-bedrock/%s/%s", m.ProviderName, m.ModelID),
		Labels:           map[string]string{},
		Provenance: map[string]string{
			"provider_name": m.ProviderName,
		},
	}
	if m.ModelLifecycleStatus != "" {
		mc.Labels["lifecycle_status"] = m.ModelLifecycleStatus
	}
	if len(m.InferenceTypesSupported) > 0 {
		mc.Labels["inference_types"] = strings.Join(m.InferenceTypesSupported, ",")
	}
	return mc
}

func bedrockCustomToModelComponent(m AWSBedrockCustomModel) *aibom.ModelComponent {
	mc := &aibom.ModelComponent{
		Name:     m.ModelName,
		Provider: "aws-bedrock",
		ModelId:  m.ModelArn,
		Purl:     fmt.Sprintf("pkg:aws-bedrock/custom/%s", m.ModelName),
		Labels:   map[string]string{},
		Provenance: map[string]string{
			"model_arn": m.ModelArn,
		},
	}
	if m.BaseModel != nil {
		mc.Provenance["base_model"] = m.BaseModel.ModelID
		mc.Author = m.BaseModel.ProviderName
	}
	return mc
}

func sagemakerToModelComponent(m AWSSageMakerModel) *aibom.ModelComponent {
	return &aibom.ModelComponent{
		Name:     m.Name,
		Provider: "aws-sagemaker",
		ModelId:  m.Arn,
		Purl:     fmt.Sprintf("pkg:aws-sagemaker/%s", m.Name),
		Labels:   map[string]string{},
		Provenance: map[string]string{
			"arn": m.Arn,
		},
	}
}

func vertexaiToModelComponent(m GCPVertexAIModel) *aibom.ModelComponent {
	return &aibom.ModelComponent{
		Name:     m.DisplayName,
		Provider: "gcp-vertexai",
		ModelId:  m.Name,
		Purl:     fmt.Sprintf("pkg:gcp-vertexai/%s", m.DisplayName),
		Labels:   map[string]string{},
		Provenance: map[string]string{
			"resource_name": m.Name,
		},
	}
}

func azureOpenAIToModelComponent(m AzureCognitiveServicesAccount) *aibom.ModelComponent {
	return &aibom.ModelComponent{
		Name:     m.Name,
		Provider: "azure-openai",
		ModelId:  m.ID,
		Labels: map[string]string{
			"location":           m.Location,
			"public_network":     m.PublicNetworkAccess,
			"disable_local_auth": fmt.Sprintf("%t", m.DisableLocalAuth),
		},
		Provenance: map[string]string{
			"endpoint":    m.Endpoint,
			"resource_id": m.ID,
		},
		SourceUrl: m.Endpoint,
	}
}

func vllmToModelComponent(v *VLLMData) *aibom.ModelComponent {
	mc := &aibom.ModelComponent{
		Name:     "vLLM Server",
		Provider: "vllm",
		Version:  v.Version,
		Labels:   map[string]string{},
		Provenance: map[string]string{
			"runtime": "vllm",
		},
	}
	if v.Server != nil {
		mc.Purl = fmt.Sprintf("pkg:vllm/server@%s", v.Server.Version)
		if v.Server.BaseUrl != "" {
			mc.SourceUrl = v.Server.BaseUrl
			mc.Labels["base_url"] = v.Server.BaseUrl
		}
		mc.Labels["tls"] = fmt.Sprintf("%t", v.Server.UsesTls)
		mc.Labels["reachable"] = fmt.Sprintf("%t", v.Server.Reachable)
		if v.Server.Version != "" {
			mc.Version = v.Server.Version
		}
	}
	return mc
}

func vllmModelToComponent(m VLLMModel, server *VLLMServer) *aibom.ModelComponent {
	mc := &aibom.ModelComponent{
		Name:     m.ID,
		Provider: "vllm",
		ModelId:  m.ID,
		Author:   m.OwnedBy,
		Labels:   map[string]string{},
		Provenance: map[string]string{
			"runtime": "vllm",
		},
	}

	if m.Root != "" && m.Root != m.ID {
		mc.SourceUrl = fmt.Sprintf("https://huggingface.co/%s", m.Root)
		mc.Provenance["root"] = m.Root
		mc.Purl = fmt.Sprintf("pkg:huggingface/%s", m.Root)
	} else {
		mc.Purl = fmt.Sprintf("pkg:vllm/%s", m.ID)
	}

	if m.Parent != "" {
		mc.Provenance["parent"] = m.Parent
		mc.ApproachType = "fine-tuned"
	}

	if m.MaxModelLen > 0 {
		mc.Provenance["context_length"] = fmt.Sprintf("%d", m.MaxModelLen)
	}

	if m.Created != "" {
		mc.CreatedAt = m.Created
	}

	if server != nil {
		if server.BaseUrl != "" {
			mc.Labels["base_url"] = server.BaseUrl
		}
		mc.Labels["tls"] = fmt.Sprintf("%t", server.UsesTls)
	}

	return mc
}

func claudeModelToComponent(m ClaudeModel) *aibom.ModelComponent {
	mc := &aibom.ModelComponent{
		Name:               m.DisplayName,
		Provider:           "anthropic",
		ModelId:            m.ID,
		Author:             m.Vendor,
		ArchitectureFamily: m.Family,
		CreatedAt:          m.CreatedAt,
		Labels:             map[string]string{},
		Provenance:         map[string]string{},
		Purl:               fmt.Sprintf("pkg:anthropic/%s", m.ID),
	}

	// Derive capabilities from supported features
	if m.ImageInputSupported {
		mc.InputModalities = append(mc.InputModalities, "image")
	}
	if m.PdfInputSupported {
		mc.InputModalities = append(mc.InputModalities, "pdf")
	}
	mc.InputModalities = append(mc.InputModalities, "text")
	mc.OutputModalities = append(mc.OutputModalities, "text")

	var caps []string
	if m.ThinkingSupported {
		caps = append(caps, "extended-thinking")
	}
	if m.CodeExecutionSupported {
		caps = append(caps, "code-execution")
	}
	if m.CitationsSupported {
		caps = append(caps, "citations")
	}
	if m.StructuredOutputsSupported {
		caps = append(caps, "structured-outputs")
	}
	if m.BatchSupported {
		caps = append(caps, "batch")
	}
	mc.Capabilities = caps

	if m.MaxInputTokens > 0 {
		mc.Provenance["max_input_tokens"] = fmt.Sprintf("%d", m.MaxInputTokens)
	}
	if m.MaxTokens > 0 {
		mc.Provenance["max_output_tokens"] = fmt.Sprintf("%d", m.MaxTokens)
	}

	return mc
}

func claudeAgentToComponent(a ClaudeAgent, skills []ClaudeSkill) *aibom.AgentComponent {
	ac := &aibom.AgentComponent{
		Name:     a.Name,
		Provider: "anthropic",
		Model:    a.Model,
		Version:  fmt.Sprintf("%d", a.Version),
		Labels: map[string]string{
			"agent_id": a.ID,
		},
	}

	if a.CreatedAt != "" {
		ac.Labels["created_at"] = a.CreatedAt
	}

	// Attach skills discovered in the same scan
	for _, s := range skills {
		ac.Skills = append(ac.Skills, &aibom.AgentSkill{
			Name:   s.DisplayTitle,
			Source: s.Source,
		})
	}

	return ac
}

func claudeMemoryStoreToKB(ms ClaudeMemoryStore) *aibom.KnowledgeBase {
	return &aibom.KnowledgeBase{
		Name:        ms.Name,
		Provider:    "anthropic",
		ID:          ms.ID,
		Description: ms.Description,
		StorageType: "memory-store",
		Labels: map[string]string{
			"created_at": ms.CreatedAt,
		},
	}
}

func openaiModelToComponent(m OpenAIModel, ftJobs []OpenAIFineTuningJob) *aibom.ModelComponent {
	mc := &aibom.ModelComponent{
		Name:       m.ID,
		Provider:   "openai",
		ModelId:    m.ID,
		Author:     m.OwnedBy,
		Labels:     map[string]string{},
		Provenance: map[string]string{},
		Purl:       fmt.Sprintf("pkg:openai/%s", m.ID),
	}

	if m.CreatedAt != "" {
		mc.CreatedAt = m.CreatedAt
	}

	if m.IsFineTuned {
		mc.ApproachType = "fine-tuned"
		if m.BaseModel != "" {
			mc.Provenance["base_model"] = m.BaseModel
		}

		// Enrich with fine-tuning job data if available
		for _, job := range ftJobs {
			if job.FineTunedModel == m.ID && job.Status == "succeeded" {
				mc.Provenance["fine_tune_job"] = job.ID
				mc.Provenance["fine_tune_base"] = job.Model
				if job.TrainedTokens > 0 {
					mc.Provenance["trained_tokens"] = fmt.Sprintf("%d", job.TrainedTokens)
				}
				break
			}
		}
	}

	return mc
}

func openaiVectorStoreToKB(vs OpenAIVectorStore) *aibom.KnowledgeBase {
	kb := &aibom.KnowledgeBase{
		Name:        vs.Name,
		Provider:    "openai",
		ID:          vs.ID,
		Status:      vs.Status,
		StorageType: "vector-store",
		Labels:      map[string]string{},
	}

	if vs.UsageBytes > 0 {
		kb.Labels["usage_bytes"] = fmt.Sprintf("%d", vs.UsageBytes)
	}

	// Extract file count summary
	if vs.FileCounts != nil {
		total := 0
		for _, v := range vs.FileCounts {
			if n, ok := v.(float64); ok {
				total += int(n)
			}
		}
		if total > 0 {
			kb.DataSources = append(kb.DataSources, fmt.Sprintf("%d files", total))
		}
	}

	return kb
}

func bedrockGuardrailToComponent(g AWSBedrockGuardrail) *aibom.Guardrail {
	gr := &aibom.Guardrail{
		Name:     g.Name,
		Provider: "aws-bedrock",
		ID:       g.ID,
		Arn:      g.Arn,
		Status:   g.Status,
		Version:  g.Version,
		Labels:   map[string]string{},
	}

	if g.ContentPolicy != nil {
		gr.Policies = append(gr.Policies, "content-filter")
	}
	if g.SensitiveInformationPolicy != nil {
		gr.Policies = append(gr.Policies, "pii-detection")
	}
	if g.TopicPolicy != nil {
		gr.Policies = append(gr.Policies, "topic-filter")
	}
	if g.WordPolicy != nil {
		gr.Policies = append(gr.Policies, "word-filter")
	}

	return gr
}

func bedrockAgentToComponent(a AWSBedrockAgent) *aibom.AgentComponent {
	ac := &aibom.AgentComponent{
		Name:     a.Name,
		Provider: "aws-bedrock",
		Model:    a.FoundationModel,
		Labels: map[string]string{
			"status": a.Status,
		},
	}
	if a.Arn != "" {
		ac.Labels["arn"] = a.Arn
	}
	if a.ID != "" {
		ac.Labels["agent_id"] = a.ID
	}

	// Extract action group dependencies (Lambda functions, APIs)
	for _, ag := range a.ActionGroups {
		agMap, ok := ag.(map[string]any)
		if !ok {
			continue
		}
		dep := &aibom.AgentDependency{Type: "action-group"}
		if name, ok := agMap["actionGroupName"].(string); ok {
			dep.Name = name
		}
		if executor, ok := agMap["actionGroupExecutor"].(map[string]any); ok {
			if lambdaArn, ok := executor["lambda"].(string); ok {
				dep.Arn = lambdaArn
			}
		}
		ac.Dependencies = append(ac.Dependencies, dep)
	}

	// Extract knowledge base dependencies
	for _, kb := range a.KnowledgeBases {
		kbMap, ok := kb.(map[string]any)
		if !ok {
			continue
		}
		dep := &aibom.AgentDependency{Type: "knowledge-base"}
		if id, ok := kbMap["knowledgeBaseId"].(string); ok {
			dep.Name = id
		}
		if desc, ok := kbMap["description"].(string); ok && dep.Name == "" {
			dep.Name = desc
		}
		ac.Dependencies = append(ac.Dependencies, dep)
	}

	return ac
}

func bedrockFlowToComponent(f AWSBedrockFlow) *aibom.AgentComponent {
	return &aibom.AgentComponent{
		Name:     f.Name,
		Provider: "aws-bedrock",
		Version:  f.Version,
		Labels: map[string]string{
			"type":   "flow",
			"status": f.Status,
			"arn":    f.Arn,
		},
	}
}

func bedrockKBToComponent(kb AWSBedrockKnowledgeBase) *aibom.KnowledgeBase {
	kbc := &aibom.KnowledgeBase{
		Name:        kb.Name,
		Provider:    "aws-bedrock",
		ID:          kb.ID,
		Arn:         kb.Arn,
		Status:      kb.Status,
		Description: kb.Description,
		Labels:      map[string]string{},
	}

	// Extract storage type from configuration
	if kb.StorageConfiguration != nil {
		if stype, ok := kb.StorageConfiguration["type"].(string); ok {
			kbc.StorageType = stype
		}
	}

	// Extract embedding model from KB configuration
	if kb.KnowledgeBaseConfiguration != nil {
		if vecConfig, ok := kb.KnowledgeBaseConfiguration["vectorKnowledgeBaseConfiguration"].(map[string]any); ok {
			if modelArn, ok := vecConfig["embeddingModelArn"].(string); ok {
				kbc.EmbeddingModel = modelArn
			}
		}
	}

	// Extract data source names/types
	for _, ds := range kb.DataSources {
		dsMap, ok := ds.(map[string]any)
		if !ok {
			continue
		}
		label := ""
		if name, ok := dsMap["name"].(string); ok {
			label = name
		} else if dsType, ok := dsMap["type"].(string); ok {
			label = dsType
		} else if dsID, ok := dsMap["dataSourceId"].(string); ok {
			label = dsID
		}
		if label != "" {
			kbc.DataSources = append(kbc.DataSources, label)
		}
	}

	return kbc
}

func modelArmorToGuardrail(t GCPModelArmorTemplate) *aibom.Guardrail {
	return &aibom.Guardrail{
		Name:     t.Name,
		Provider: "gcp-model-armor",
		Labels:   t.Labels,
	}
}

func modelArmorFloorToGuardrail(f *GCPModelArmorFloorSetting) *aibom.Guardrail {
	g := &aibom.Guardrail{
		Name:     f.Name,
		Provider: "gcp-model-armor",
		Labels: map[string]string{
			"type": "floor-setting",
		},
	}
	if f.EnableFloorSettingEnforcement {
		g.Status = "enforced"
	}
	return g
}

func gcpCloudFuncToComponent(fn GCPCloudFunction) *aibom.AgentComponent {
	ac := &aibom.AgentComponent{
		Name:     fn.Name,
		Provider: "gcp-cloud-functions",
		Labels: map[string]string{
			"state":       fn.State,
			"environment": fn.Environment,
		},
	}
	if fn.URL != "" {
		ac.Labels["url"] = fn.URL
	}
	if fn.BuildConfig != nil {
		dep := &aibom.AgentDependency{
			Type:    "cloud-function",
			Name:    fn.Name,
			Runtime: fn.BuildConfig.Runtime,
			Labels: map[string]string{
				"entry_point": fn.BuildConfig.EntryPoint,
			},
		}
		if fn.BuildConfig.DockerRepository != "" {
			dep.ImageUri = fn.BuildConfig.DockerRepository
		}
		ac.Dependencies = append(ac.Dependencies, dep)
	}
	return ac
}

func codingAgentToComponent(name string, a *CodingAgent) *aibom.AgentComponent {
	ac := &aibom.AgentComponent{
		Name:       name,
		Provider:   "local",
		ConfigPath: a.ConfigPath,
		Version:    a.Version,
		Model:      a.Model,
		Labels:     map[string]string{},
	}

	for _, s := range a.Skills {
		ac.Skills = append(ac.Skills, &aibom.AgentSkill{
			Name:        s.Name,
			Description: s.Description,
			Source:      s.Source,
			Sha256:      s.Sha256,
		})
	}

	for _, m := range a.McpServers {
		ms := &aibom.McpServer{
			Name:    m.Name,
			Type:    m.Type,
			Command: m.Command,
			Args:    m.Args,
			Url:     m.Url,
			HasEnv:  m.HasEnv,
			Purl:    mcpServerToPurl(m),
		}
		ac.McpServers = append(ac.McpServers, ms)
	}

	for _, p := range a.Plugins {
		ap := &aibom.AgentPlugin{
			Name:         p.Name,
			Version:      p.Version,
			Author:       p.Author,
			Description:  p.Description,
			InstallPath:  p.InstallPath,
			GitCommitSha: p.GitCommitSha,
			Enabled:      p.Enabled,
		}
		if p.GitCommitSha != "" {
			ap.Purl = fmt.Sprintf("pkg:generic/%s/%s@%s?vcs_url=git+%s", name, p.Name, p.Version, p.GitCommitSha)
		} else if p.Version != "" {
			ap.Purl = fmt.Sprintf("pkg:generic/%s/%s@%s", name, p.Name, p.Version)
		}
		ac.Plugins = append(ac.Plugins, ap)
	}

	for _, e := range a.Extensions {
		ac.Extensions = append(ac.Extensions, &aibom.AgentExtension{
			Name:        e.Name,
			Type:        e.Type,
			Description: e.Description,
			Enabled:     e.Enabled,
			Bundled:     e.Bundled,
		})
	}

	if a.Email != "" {
		ac.Labels["email"] = a.Email
	}
	if a.Organization != "" {
		ac.Labels["organization"] = a.Organization
	}
	if a.Subscription != "" {
		ac.Labels["subscription"] = a.Subscription
	}
	if a.Provider != "" {
		ac.Labels["ai_provider"] = a.Provider
	}

	return ac
}

func mcpServerToPurl(m CodingAgentMcp) string {
	if m.Command == "" && m.Url == "" {
		return ""
	}
	cmd := m.Command
	args := m.Args

	switch cmd {
	case "npx", "bunx":
		if len(args) > 0 {
			pkg := args[0]
			if pkg == "-y" && len(args) > 1 {
				pkg = args[1]
			}
			return fmt.Sprintf("pkg:npm/%s", pkg)
		}
	case "uvx", "pipx":
		if len(args) > 0 {
			return fmt.Sprintf("pkg:pypi/%s", args[0])
		}
	case "docker":
		if len(args) >= 2 && args[0] == "run" {
			return fmt.Sprintf("pkg:docker/%s", args[len(args)-1])
		}
	}
	return ""
}
