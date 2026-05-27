// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnspec/v13/cli/reporter"
	"go.mondoo.com/cnspec/v13/internal/aibom"
	"go.mondoo.com/cnspec/v13/internal/aibom/generator"
	"go.mondoo.com/cnspec/v13/internal/aibom/pack"
	"go.mondoo.com/cnspec/v13/policy/scan"
	"go.mondoo.com/mql/v13/logger"
	"go.mondoo.com/mql/v13/providers"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

func init() {
	rootCmd.AddCommand(aibomCmd)
	aibomCmd.Flags().String("asset-name", "", "User-override for the asset name")
	aibomCmd.Flags().StringToString("annotation", nil, "Add an annotation to the asset in the form KEY=VALUE")
	aibomCmd.Flags().StringP("output", "o", "markdown", "Set output format: "+aibom.AllFormats())
	aibomCmd.Flags().String("output-target", "", "Set output target to which the AIBOM report will be written")
}

var aibomCmd = &cobra.Command{
	Use:   "aibom",
	Short: "Generate an AI bill of materials (AIBOM) for AI models across providers",
	Long: `Generate an AI bill of materials (AIBOM) that inventories AI/ML models
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
`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlag("output", cmd.Flags().Lookup("output")); err != nil {
			log.Fatal().Err(err).Msg("failed to bind output flag")
		}
		if err := viper.BindPFlag("output-target", cmd.Flags().Lookup("output-target")); err != nil {
			log.Fatal().Err(err).Msg("failed to bind output-target flag")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {},
}

var aibomCmdRun = func(cmd *cobra.Command, runtime *providers.Runtime, cliRes *plugin.ParseCLIRes) {
	pb, err := pack.QueryPack()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load AIBOM query pack")
	}

	conf, err := getCobraScanConfig(cmd, runtime, cliRes)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get scan config")
	}

	conf.PolicyNames = nil
	conf.PolicyPaths = nil
	conf.Bundle = pb
	conf.IsIncognito = true

	report, err := RunScan(conf, scan.DisableProgressBar())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to run scan")
	}

	cnspecReport, err := reporter.ConvertToProto(report)
	if err == nil {
		log.Debug().Msg("converted report to proto")
		data, _ := cnspecReport.ToJSON()
		logger.DebugDumpJSON("mondoo-aibom-report", data)
	}

	boms := generator.GenerateAiBom(cnspecReport.ToCnqueryReport())

	output := viper.GetString("output")
	formatter := aibom.NewFormatter(output)
	if formatter == nil {
		log.Fatal().Msg("unsupported output format: " + output)
	}

	outputTarget := viper.GetString("output-target")
	for i := range boms {
		bom := boms[i]
		buf := bytes.Buffer{}
		err := formatter.Render(&buf, bom)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to render AIBOM")
		}

		if outputTarget != "" {
			filename := outputTarget
			if len(boms) > 1 {
				filename = fmt.Sprintf("%s-%d.%s", path.Base(outputTarget), i, path.Ext(outputTarget))
			}
			if err := os.WriteFile(filename, buf.Bytes(), 0o600); err != nil {
				log.Fatal().Err(err).Msg("failed to write AIBOM to file")
			}
		} else {
			fmt.Println(buf.String())
		}
	}
}
