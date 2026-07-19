// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnspec/v13/upload"
	rc "go.mondoo.com/cnspec/v13/upload/report_conversion"
	_ "go.mondoo.com/cnspec/v13/upload/report_conversion/all"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/fex"
	"go.mondoo.com/mql/v13/sbom"
)

// maxReportSize caps the report file read into memory (512 MiB).
const maxReportSize = 512 << 20

// sbomFormats are handled by the SBOM path: the file is decoded into an SBOM and
// scanned by Mondoo Platform (ExtendedVulnMgmt.ScanUploadedSbom), which returns
// VEX. The mql multi-decoder auto-detects the concrete encoding, so these names
// are for discoverability; all route through the same path.
var sbomFormats = map[string]bool{
	"sbom":      true,
	"cyclonedx": true,
	"spdx":      true,
}

// allFormats lists every accepted --format value (pure converters + SBOM).
func allFormats() []string {
	out := rc.Formats()
	for f := range sbomFormats {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

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
		for _, f := range allFormats() {
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

	conv, isConverter := rc.Get(format)
	if !isConverter && !sbomFormats[format] {
		return fmt.Errorf("unknown format %q; use --format list to see supported formats", format)
	}

	// Guard against reading an accidentally huge file into memory (arbitrary
	// user-supplied path).
	if info, err := os.Stat(args[0]); err != nil {
		return fmt.Errorf("stat %s: %w", args[0], err)
	} else if info.Size() > maxReportSize {
		return fmt.Errorf("report %s is %d bytes, which exceeds the %d byte limit", args[0], info.Size(), int64(maxReportSize))
	}

	data, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("read %s: %w", args[0], err)
	}

	configPath, _ := cmd.Flags().GetString("config")
	space, _ := cmd.Flags().GetString("space")
	opts := upload.Opts{ConfigPath: configPath, ScopeMrn: space}

	var docs []*fex.FindingDocument
	if sbomFormats[format] {
		// SBOM path: decode → scan on Mondoo Platform (returns VEX) → documents.
		docs, err = scanSBOMFile(cmd.Context(), opts, data)
	} else {
		docs, err = conv(data)
	}
	if err != nil {
		if upload.IsNoCredentials(err) {
			return fmt.Errorf("no Mondoo credentials found; run `cnspec login` or pass --config <path>")
		}
		return fmt.Errorf("process %s: %w", args[0], err)
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

	err = upload.UploadFindings(cmd.Context(), opts, docs, source)
	if err != nil {
		if upload.IsNoCredentials(err) {
			return fmt.Errorf("no Mondoo credentials found; run `cnspec login` or pass --config <path>")
		}
		return fmt.Errorf("upload findings: %w", err)
	}

	fmt.Printf("Uploaded %d finding(s) from %s\n", len(docs), args[0])
	return nil
}

// scanSBOMFile decodes SBOM bytes (CycloneDX/SPDX/Mondoo JSON, auto-detected),
// scans them on Mondoo Platform, and wraps the resulting VEX as documents.
func scanSBOMFile(ctx context.Context, opts upload.Opts, data []byte) ([]*fex.FindingDocument, error) {
	bom, err := sbom.DefaultMultiDecoder().Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parse SBOM (expected CycloneDX, SPDX, or Mondoo JSON): %w", err)
	}
	vex, err := upload.ScanSBOM(ctx, opts, bom)
	if err != nil {
		return nil, err
	}
	return fex.VexToDocuments(vex), nil
}
