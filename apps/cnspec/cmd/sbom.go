// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnspec/cli/reporter"
	"go.mondoo.com/cnspec/internal/sbom/pack"
	"go.mondoo.com/cnspec/internal/scandump"
	"go.mondoo.com/mql/providers"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/sbom"
	"go.mondoo.com/mql/sbom/generator"
)

func init() {
	rootCmd.AddCommand(sbomCmd)
	sbomCmd.Flags().String("asset-name", "", "User-override for the asset name")
	sbomCmd.Flags().StringToString("annotation", nil, "Add an annotation to the asset in the form KEY=VALUE") // user-added, editable
	sbomCmd.Flags().StringP("output", "o", "list", "Set output format: "+sbom.AllFormats())
	sbomCmd.Flags().String("output-target", "", "Set output target to which the SBOM report will be written")
	sbomCmd.Flags().Bool("with-evidence", false, "Include evidence for each component")
	sbomCmd.Flags().Bool("with-cpes", false, "Generate CPEs for each component")
	addInventoryFlags(sbomCmd, "Set the path to the inventory file. With several assets, --output-target gets one file per asset")
}

var sbomCmd = &cobra.Command{
	Use:   "sbom",
	Short: "Experimental: Generate a software bill of materials (SBOM) for a given asset",
	Long: `Generate a software bill of materials (SBOM) for a given asset. The SBOM
is a representation of the asset's software components and their dependencies.

The following formats are supported:
- list (default)
- cnquery-json
- cyclonedx-json
- cyclonedx-xml
- spdx-json
- spdx-tag-value

To generate SBOMs for the assets of an inventory file, with the credentials
it defines, use --inventory-file instead of a provider subcommand:

  cnspec sbom --inventory-file inventory.yml -o cyclonedx-json --output-target sbom.json

An inventory with several assets produces one SBOM per asset. With
--output-target, each is written to its own file, with the index inserted
before the extension (sbom-0.json, sbom-1.json, ...), ordered by asset name.
Without it, they are printed one after another. Assets that could not be
scanned get no SBOM; they are listed and the command exits 1 after writing
the others.

Note this command is experimental and may change in the future.
`,
	PreRun: func(cmd *cobra.Command, args []string) {
		bindInventoryFlags(cmd)

		err := viper.BindPFlag("output", cmd.Flags().Lookup("output"))
		if err != nil {
			log.Fatal().Err(err).Msg("failed to bind output flag")
		}

		err = viper.BindPFlag("output-target", cmd.Flags().Lookup("output-target"))
		if err != nil {
			log.Fatal().Err(err).Msg("failed to bind output-target flag")
		}

		err = viper.BindPFlag("with-evidence", cmd.Flags().Lookup("with-evidence"))
		if err != nil {
			log.Fatal().Err(err).Msg("failed to bind with-evidence flag")
		}

		err = viper.BindPFlag("with-cpes", cmd.Flags().Lookup("with-cpes"))
		if err != nil {
			log.Fatal().Err(err).Msg("failed to bind with-cpes flag")
		}

		// sbom.New falls back to the table format instead of returning nil, so the
		// format has to be rejected here or an unsupported -o silently renders a table
		if format := viper.GetString("output"); !sbom.IsSupportedFormat(format) {
			log.Fatal().Msg("unsupported output format: " + format + ". Supported formats: " + sbom.AllFormats())
		}
	},
	// we have to initialize an empty run so it shows up as a runnable command in --help
	Run: func(cmd *cobra.Command, args []string) {},
}

var sbomCmdRun = func(cmd *cobra.Command, runtime *providers.Runtime, cliRes *plugin.ParseCLIRes) {
	log.Info().Msg("This command is experimental. Please report any issues to https://github.com/mondoohq/cnspec.")

	ctx := setupDebugDumps(context.Background(), "cnspec-sbom-debug")

	pb, err := pack.QueryPack()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load query pack")
	}

	conf, err := getCobraScanConfig(cmd, runtime, cliRes)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get scan config")
	}

	conf.PolicyNames = nil
	conf.PolicyPaths = nil
	conf.Bundle = pb
	conf.IsIncognito = true

	report, err := RunScan(ctx, conf)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to run scan")
	}

	cnspecReport, err := reporter.ConvertToProto(report)
	if err == nil {
		log.Debug().Msg("converted report to proto")
		data, _ := cnspecReport.ToJSON()
		scandump.JSON(ctx, "sbom-report", data)
	}

	// Assets that could not be scanned get no bill of materials; they are
	// reported, and fail the command, after the others are written.
	collected, failures := withoutFailedAssets(cnspecReport.ToCnqueryReport())
	boms := []*sbom.Sbom{}
	for _, bom := range generator.GenerateBom(collected) {
		// A failed asset has no packages. Rendering it would write a valid but
		// empty SBOM that is indistinguishable from an asset with no software.
		if bom.Status == sbom.Status_STATUS_FAILED {
			failures = append(failures, bomFailure{Asset: bom.GetAsset().GetName(), Reason: bom.ErrorMessage})
			continue
		}
		boms = append(boms, bom)
	}
	// Sorted after the failed assets are dropped, so the index in each output
	// file name counts only the documents that are written.
	sortBomsByAssetName(boms, func(b *sbom.Sbom) string { return b.GetAsset().GetName() })

	// the output format is validated in PreRun, sbom.New always returns a handler
	exporter := sbom.New(viper.GetString("output"))

	if viper.GetBool("with-evidence") {
		exporter.ApplyOptions(sbom.WithEvidence())
	}

	if viper.GetBool("with-cpes") {
		exporter.ApplyOptions(sbom.WithCPE())
	}

	outputTarget := viper.GetString("output-target")
	if len(boms) > 1 && outputTarget == "" {
		log.Warn().Int("assets", len(boms)).Msg("printing one SBOM per asset; use --output-target to write each to its own file")
	}
	for i := range boms {
		bom := boms[i]
		output := bytes.Buffer{}
		err := exporter.Render(&output, bom)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to render SBOM")
		}

		if outputTarget != "" {
			filename := bomOutputFile(outputTarget, i, len(boms))
			err := os.WriteFile(filename, output.Bytes(), 0o600)
			if err != nil {
				log.Fatal().Err(err).Msg("failed to write SBOM to file")
			}
		} else {
			fmt.Println(output.String())
		}
	}

	if err := bomFailuresError("SBOM", failures, len(boms)); err != nil {
		log.Fatal().Msg(err.Error())
	}
}
