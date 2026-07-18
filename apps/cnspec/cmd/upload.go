// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnspec/v13/upload"
	rc "go.mondoo.com/cnspec/v13/upload/report_conversion"
	_ "go.mondoo.com/cnspec/v13/upload/report_conversion/all"
)

func init() {
	rootCmd.AddCommand(uploadCmd)
	uploadCmd.Flags().String("format", "", "Source report format (use --format list to see all)")
	uploadCmd.Flags().String("source", "", "Producing-tool name recorded on findings (default: the --format value)")
	uploadCmd.Flags().String("space", "", "Target space MRN (default: from the Mondoo config)")
	uploadCmd.Flags().String("config", "", "Path to the Mondoo config / service account")
	uploadCmd.Flags().Bool("dry-run", false, "Convert and summarize, but do not upload")
}

var uploadCmd = &cobra.Command{
	Use: "upload --format <format> [flags] <file>",
	// Hidden until validated in production; the command works but is not yet
	// advertised in `cnspec --help`.
	Hidden: true,
	Short:  "Experimental: Upload a third-party scan report to Mondoo Platform as findings",
	Long: `Convert a third-party scanner's report file into Mondoo findings (FEX/VEX)
and upload them to Mondoo Platform.

Use --format list to see the supported formats. Standard open formats
(e.g. SARIF) are converted locally by cnspec; other formats are converted by the
Mondoo Platform after upload.

Example:
  cnspec upload --format sarif results.sarif
`,
	Args: cobra.MaximumNArgs(1),
	RunE: runUpload,
}

func runUpload(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")

	if format == "list" {
		fmt.Println("Supported formats:")
		for _, f := range rc.Formats() {
			fmt.Println("  " + f)
		}
		return nil
	}
	if format == "" {
		return fmt.Errorf("--format is required (use --format list to see supported formats)")
	}
	if len(args) != 1 {
		return fmt.Errorf("exactly one report file is required")
	}

	conv, ok := rc.Get(format)
	if !ok {
		return fmt.Errorf("unknown format %q; use --format list to see supported formats", format)
	}

	data, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("read %s: %w", args[0], err)
	}
	docs, err := conv(data)
	if err != nil {
		return fmt.Errorf("convert %s: %w", args[0], err)
	}
	if len(docs) == 0 {
		fmt.Printf("No findings in %s\n", args[0])
		return nil
	}
	for i, d := range docs {
		if verr := rc.Validate(d); verr != nil {
			log.Warn().Msgf("finding %d is not clean and may be rejected: %v", i, verr)
		}
	}

	if dryRun, _ := cmd.Flags().GetBool("dry-run"); dryRun {
		fmt.Printf("Converted %d finding(s) from %s (dry run — not uploaded)\n", len(docs), args[0])
		return nil
	}

	source, _ := cmd.Flags().GetString("source")
	if source == "" {
		source = format
	}
	configPath, _ := cmd.Flags().GetString("config")
	space, _ := cmd.Flags().GetString("space")

	err = upload.UploadFindings(context.Background(), upload.Opts{ConfigPath: configPath, ScopeMrn: space}, docs, source)
	if err != nil {
		if upload.IsNoCredentials(err) {
			return fmt.Errorf("no Mondoo credentials found; run `cnspec login` or pass --config <path>")
		}
		return fmt.Errorf("upload findings: %w", err)
	}

	fmt.Printf("Uploaded %d finding(s) from %s\n", len(docs), args[0])
	return nil
}
