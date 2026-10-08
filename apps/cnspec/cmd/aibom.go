// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnspec/cli/reporter"
	"go.mondoo.com/cnspec/internal/aibom"
	"go.mondoo.com/cnspec/internal/aibom/generator"
	"go.mondoo.com/cnspec/internal/aibom/pack"
	"go.mondoo.com/cnspec/policy/scan"
	"go.mondoo.com/mql/logger"
	"go.mondoo.com/mql/providers"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

func init() {
	rootCmd.AddCommand(aibomCmd)
	aibomCmd.Flags().String("asset-name", "", "User-override for the asset name")
	aibomCmd.Flags().StringToString("annotation", nil, "Add an annotation to the asset in the form KEY=VALUE")
	aibomCmd.Flags().StringP("output", "o", "markdown", "Set output format: "+aibom.AllFormats())
	aibomCmd.Flags().String("output-target", "", "Set output target to which the AIBOM report will be written")
	addInventoryFlags(aibomCmd, "Set the path to the inventory file. With several assets, --output-target gets one file per asset")
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

To generate AIBOMs for the assets of an inventory file, with the credentials
it defines, use --inventory-file instead of a provider subcommand. An
inventory with several assets produces one AIBOM per asset. With
--output-target, each is written to its own file, with the index inserted
before the extension (aibom-0.json, aibom-1.json, ...), ordered by asset name.
Assets that could not be scanned get no AIBOM; they are listed and the
command exits 1 after writing the others.

Examples:
  cnspec aibom local
  cnspec aibom local -o json
  cnspec aibom ollama -o cyclonedx-json
  cnspec aibom aws -o cyclonedx-json
  cnspec aibom --inventory-file inventory.yml -o json --output-target aibom.json
`,
	PreRun: func(cmd *cobra.Command, args []string) {
		bindInventoryFlags(cmd)

		if err := viper.BindPFlag("output", cmd.Flags().Lookup("output")); err != nil {
			log.Fatal().Err(err).Msg("failed to bind output flag")
		}
		if err := viper.BindPFlag("output-target", cmd.Flags().Lookup("output-target")); err != nil {
			log.Fatal().Err(err).Msg("failed to bind output-target flag")
		}

		// aibom.NewFormatter falls back to markdown instead of returning nil, so the
		// format has to be rejected here or an unsupported -o silently renders markdown
		if format := viper.GetString("output"); !aibom.IsSupportedFormat(format) {
			log.Fatal().Msg("unsupported output format: " + format + ". Supported formats: " + aibom.AllFormats())
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

	ctx := setupDebugDumps(context.Background(), "cnspec-aibom-debug")

	report, err := RunScan(ctx, conf, scan.DisableProgressBar())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to run scan")
	}

	cnspecReport, err := reporter.ConvertToProto(report)
	if err == nil {
		log.Debug().Msg("converted report to proto")
		data, _ := cnspecReport.ToJSON()
		logger.DebugDumpJSON("mondoo-aibom-report", data)
	}

	// Assets that could not be scanned get no bill of materials; they are
	// reported, and fail the command, after the others are written.
	collected, failures := withoutFailedAssets(cnspecReport.ToCnqueryReport())
	boms := []*aibom.AiBom{}
	for _, bom := range generator.GenerateAiBom(collected) {
		// A failed asset has no data. Rendering it would write an AIBOM that
		// looks like an asset without any AI usage.
		if bom.Status == aibom.Status_STATUS_FAILED {
			failures = append(failures, bomFailure{Asset: aibomAssetName(bom), Reason: strings.Join(bom.Errors, "; ")})
			continue
		}
		boms = append(boms, bom)
	}
	// Sorted after the failed assets are dropped, so the index in each output
	// file name counts only the documents that are written.
	sortBomsByAssetName(boms, aibomAssetName)

	// the output format is validated in PreRun, aibom.NewFormatter always returns a handler
	formatter := aibom.NewFormatter(viper.GetString("output"))

	outputTarget := viper.GetString("output-target")
	if len(boms) > 1 && outputTarget == "" {
		log.Warn().Int("assets", len(boms)).Msg("printing one AIBOM per asset; use --output-target to write each to its own file")
	}
	for i := range boms {
		bom := boms[i]
		buf := bytes.Buffer{}
		err := formatter.Render(&buf, bom)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to render AIBOM")
		}

		if outputTarget != "" {
			filename := bomOutputFile(outputTarget, i, len(boms))
			if err := os.WriteFile(filename, buf.Bytes(), 0o600); err != nil {
				log.Fatal().Err(err).Msg("failed to write AIBOM to file")
			}
		} else {
			fmt.Println(buf.String())
		}
	}

	if err := bomFailuresError("AIBOM", failures, len(boms)); err != nil {
		log.Fatal().Msg(err.Error())
	}
}

// aibomAssetName is the name of the asset an AIBOM describes, or "" if it has
// none.
func aibomAssetName(b *aibom.AiBom) string {
	if b == nil || b.Asset == nil {
		return ""
	}
	return b.Asset.Name
}
