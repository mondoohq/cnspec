---
id: cnspec_aibom
title: cnspec aibom
---


Generate an AI bill of materials (AIBOM) for AI models across providers

### Synopsis

Generate an AI bill of materials (AIBOM) that inventories AI/ML models
across cloud providers, model registries, inference APIs, and local runtimes.

Supported providers:
- local          Local system (agents, cached models)
- ollama         Ollama models
- huggingface    HuggingFace Hub models
- openai         OpenAI API (models, vector stores, fine-tuning)
- claude         Anthropic Claude API (models, agents, skills)
- vllm           vLLM inference server
- aws            AWS Bedrock + SageMaker
- gcp            GCP Vertex AI + Model Armor
- azure          Azure AI Services (OpenAI, Cognitive Services)

Output formats:
- markdown (default)
- json
- cyclonedx-json
- cyclonedx-xml

Examples:
  cnspec aibom local
  cnspec aibom local -o json
  cnspec aibom ollama -o cyclonedx-json
  cnspec aibom aws -o cyclonedx-json


```bash
cnspec aibom [flags]
```

### Options

```
      --annotation stringToString   Add an annotation to the asset in the form KEY=VALUE (default [])
      --asset-name string           User-override for the asset name
  -h, --help                        help for aibom
  -o, --output string               Set output format: markdown, json, cyclonedx-json, cyclonedx-xml (default "markdown")
      --output-target string        Set output target to which the AIBOM report will be written
```

### Options inherited from parent commands

```
      --api-proxy string        Set the proxy for communications with Mondoo Platform API
      --auto-update             Enable automatic provider installation and update (default true)
      --config string           Set config file path (default $HOME/.config/mondoo/mondoo.yml)
      --log-level string        Set the log level: error, warn, info, debug, trace (default "info")
      --logging-config string   Path to a logging configuration file (YAML or JSON) that selects the log writer, level, and writer-specific options
      --strict                  Default MQL strict mode for policies that do not declare one: every link in an MQL chain must resolve
  -v, --verbose                 Enable verbose output
```

### SEE ALSO

* [cnspec](cnspec.md)	 - cnspec CLI

