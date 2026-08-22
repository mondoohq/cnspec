// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"errors"
	"sort"
	"strings"

	"github.com/muesli/termenv"
	"go.mondoo.com/cnspec/cli/reporter/ocsf"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

type Format byte

type PrintConfig struct {
	format               Format
	isCompact            bool
	printControls        bool
	printChecks          bool
	printData            bool
	printRisks           bool
	printVulnerabilities bool
	// detailed enables rich per-check output (description, query, assessment,
	// remediation, references). Currently consumed by the JUnit reporter.
	detailed bool
	// ocsfVersion is the OCSF schema version the ocsf-json and ocsf-parquet
	// formats emit.
	ocsfVersion ocsf.Version
	// ocsfFindings selects which OCSF class check results are reported as.
	ocsfFindings OcsfFindingClasses
}

// ocsfConfig collects the OCSF settings the converter needs.
func (p *PrintConfig) ocsfConfig() ocsfConfig {
	return ocsfConfig{
		version:     p.ocsfVersion,
		findings:    p.ocsfFindings,
		includeData: p.printData,
	}
}

func defaultPrintConfig() *PrintConfig {
	return &PrintConfig{
		format:               FormatCompact,
		isCompact:            true,
		printControls:        false,
		printChecks:          true,
		printData:            false,
		printRisks:           true,
		printVulnerabilities: true,
		ocsfVersion:          ocsf.DefaultVersion,
		ocsfFindings:         OcsfFindingsCompliance,
	}
}

const (
	OptionPrintChecks   = "checks"
	OptionPrintControls = "controls"
	OptionPrintData     = "data"
	OptionPrintRisks    = "risks"
	OptionPrintVulns    = "vulns"
	OptionDetailed      = "detailed"

	// OptionOcsfVersion selects the OCSF schema version, e.g.
	// "--output ocsf-json,ocsf-version=1.9.0".
	OptionOcsfVersion = "ocsf-version"

	// OptionOcsfFindings selects the OCSF class check results are reported as:
	// compliance (the default), detection, or both.
	OptionOcsfFindings = "ocsf-findings"
)

func ParseConfig[T string | Format](raw T) (*PrintConfig, error) {
	res := defaultPrintConfig()
	if string(raw) == "" {
		return res, nil
	}

	parts := strings.Split(string(raw), ",")
	var unknown []string
	for _, cmd := range parts {
		cur := strings.ToLower(cmd)

		format, ok := Formats[cur]
		if ok {
			res.setFormat(format)
			continue
		}

		switch cur {
		case OptionPrintControls:
			res.printControls = true
		case OptionPrintChecks:
			res.printChecks = true
		case OptionPrintData:
			res.printData = true
		case OptionPrintRisks, "risk":
			res.printRisks = true
		case OptionPrintVulns, "vuln":
			res.printVulnerabilities = true
		case "no" + OptionPrintControls:
			res.printControls = false
		case "no" + OptionPrintChecks:
			res.printChecks = false
		case "no" + OptionPrintData:
			res.printData = false
		case "no" + OptionPrintRisks, "norisk":
			res.printRisks = false
		case "no" + OptionPrintVulns, "novuln":
			res.printVulnerabilities = false
		case OptionDetailed:
			res.detailed = true
		case "no" + OptionDetailed:
			res.detailed = false
		default:
			key, value, isPair := strings.Cut(cur, "=")
			if isPair && key == OptionOcsfVersion {
				version, err := ocsf.ParseVersion(value)
				if err != nil {
					return res, err
				}
				res.ocsfVersion = version
				continue
			}
			if isPair && key == OptionOcsfFindings {
				findings, err := parseOcsfFindings(value)
				if err != nil {
					return res, err
				}
				res.ocsfFindings = findings
				continue
			}
			unknown = append(unknown, cur)
		}
	}

	if len(unknown) != 0 {
		return res, errors.New("unknown terms entered: " + strings.Join(unknown, ", ") + ". " + AllAvailableOptions())
	}
	return res, nil
}

// parseOcsfFindings resolves the ocsf-findings option.
func parseOcsfFindings(raw string) (OcsfFindingClasses, error) {
	switch strings.TrimSpace(raw) {
	case "", "compliance":
		return OcsfFindingsCompliance, nil
	case "detection":
		return OcsfFindingsDetection, nil
	case "both":
		return OcsfFindingsBoth, nil
	default:
		return 0, errors.New("unknown " + OptionOcsfFindings + " value " + raw +
			", expected one of: compliance, detection, both")
	}
}

func AllAvailableOptions() string {
	return "Available output formats: " + AllFormats() + ".\n" +
		"Available options: " + AllOptions() + ".\n" +
		"Combine with commas, example: compact,nodata,nocontrols"
}

func (p *PrintConfig) setFormat(f Format) *PrintConfig {
	p.format = f
	switch f {
	case FormatCompact:
		p.isCompact = true
	case FormatSummary:
		p.isCompact = true
		p.printChecks = false
		p.printControls = false
		p.printData = false
		p.printRisks = false
		p.printVulnerabilities = false
	case FormatFull:
		p.isCompact = false
		p.printChecks = true
		p.printControls = true
		p.printData = true
		p.printRisks = true
		p.printVulnerabilities = true
	}
	return p
}

func (p *PrintConfig) printContents() bool {
	return p.printControls || p.printChecks || p.printData || p.printVulnerabilities || p.printRisks
}

const (
	FormatCompact Format = iota + 1
	FormatSummary
	FormatFull
	FormatReport
	FormatYAMLv1
	FormatJSONv1
	FormatJUnit
	FormatCSV
	FormatJSONv2
	FormatYAMLv2
	FormatSarif
	FormatOcsfJson
	FormatOcsfParquet
)

// Formats that are supported by the reporter
var Formats = map[string]Format{
	"compact": FormatCompact,
	"summary": FormatSummary,
	"full":    FormatFull,
	"":        FormatCompact,
	"report":  FormatReport,
	"yaml-v1": FormatYAMLv1,
	"yaml-v2": FormatYAMLv2,
	"yaml":    FormatYAMLv1,
	"yml":     FormatYAMLv2,
	"json-v1": FormatJSONv1,
	"json-v2": FormatJSONv2,
	"json":    FormatJSONv2,
	"junit":   FormatJUnit,
	"csv":     FormatCSV,
	"sarif":   FormatSarif,
	// OCSF, for security data lakes. "ocsf" is an alias for the JSON flavor.
	"ocsf":         FormatOcsfJson,
	"ocsf-json":    FormatOcsfJson,
	"ocsf-parquet": FormatOcsfParquet,
}

func AllFormats() string {
	var res []string
	for k := range Formats {
		if k != "" && // default if nothing is provided, ignore
			k != "yml" && // don't show both yaml and yml
			k != "ocsf" { // don't show both ocsf and ocsf-json
			res = append(res, k)
		}
	}
	sort.Strings(res)
	return strings.Join(res, ", ")
}

func AllOptions() string {
	return "[no]" + OptionPrintChecks + ", " +
		"[no]" + OptionPrintControls + ", " +
		"[no]" + OptionPrintData + ", " +
		"[no]" + OptionPrintRisks + ", " +
		"[no]" + OptionPrintVulns + ", " +
		"[no]" + OptionDetailed + " (junit), " +
		OptionOcsfVersion + "=" + strings.Join(ocsf.SupportedVersions(), "|") + " (ocsf), " +
		OptionOcsfFindings + "=compliance|detection|both (ocsf)"
}

func (r *Reporter) scoreColored(rating policy.ScoreRating, s string) string {
	switch rating {
	case policy.ScoreRating_aPlus, policy.ScoreRating_a, policy.ScoreRating_aMinus:
		return termenv.String(s).Foreground(r.Colors.Good).String()
	case policy.ScoreRating_bPlus, policy.ScoreRating_b, policy.ScoreRating_bMinus:
		return termenv.String(s).Foreground(r.Colors.Low).String()
	case policy.ScoreRating_cPlus, policy.ScoreRating_c, policy.ScoreRating_cMinus:
		return termenv.String(s).Foreground(r.Colors.Medium).String()
	case policy.ScoreRating_dPlus, policy.ScoreRating_d, policy.ScoreRating_dMinus:
		return termenv.String(s).Foreground(r.Colors.High).String()
	case policy.ScoreRating_failed:
		return termenv.String(s).Foreground(r.Colors.Critical).String()
	}
	return s
}

func getPlatformNameForAsset(asset *inventory.Asset) string {
	platformName := ""
	if asset.Platform != nil {
		if asset.Platform.Title == "" {
			platformName = asset.Platform.Name
		} else {
			platformName = asset.Platform.Title
		}
	}
	return platformName
}
