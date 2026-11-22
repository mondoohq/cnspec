// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package tofex

import (
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnspec/reports/ocsf"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/fex"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Property keys of a finding.
const (
	// PropertyPrefix namespaces every OCSF value that fex has no field for.
	PropertyPrefix = "ocsf/"
	// PropertySource names where in the OCSF finding an evidence came from.
	PropertySource = PropertyPrefix + "source"

	// PropertyTactics and PropertyTechniques hold every ATT&CK id of a finding,
	// comma separated. An evidence names one tactic and one technique.
	PropertyTactics    = "attack.mitre.org/tactics"
	PropertyTechniques = "attack.mitre.org/techniques"
)

// OCSF status ids that only exist from 1.4 on, so the generated enums, which
// take their values from the oldest supported version, have no constant for
// them.
const (
	statusArchived = 5
	statusDeleted  = 6
)

// Options control the parts of the mapping a caller may know better than the
// finding itself.
type Options struct {
	// Source names the source of the finding. Empty takes the product name from
	// the finding's metadata.
	Source string
	// Category of the finding. Unspecified is CATEGORY_THREAT.
	Category fex.FindingDetail_Category
	// Affects replaces the affected components. Nil derives one component from
	// the finding's device.
	Affects []*fex.Affects
}

// DetectionFinding maps an OCSF Detection Finding (class 2004) onto fex.
//
// It fails when the finding has no finding_info.uid, which is the fex id, or
// when neither Options.Source nor metadata.product.name names a source.
func DetectionFinding(f *ocsf.DetectionFinding, opts Options) (*fex.FindingExchange, error) {
	if f == nil {
		return nil, errors.New("tofex: nil detection finding")
	}
	if f.ClassUID != 0 && f.ClassUID != ocsf.ClassUIDDetectionFinding {
		return nil, errors.Newf("tofex: class %d is not a Detection Finding", f.ClassUID)
	}
	info := f.FindingInfo
	if info.UID == "" {
		return nil, errors.New("tofex: detection finding has no finding_info.uid")
	}

	source := opts.Source
	if source == "" {
		source = f.Metadata.Product.Name
	}
	if source == "" {
		return nil, errors.New("tofex: no source, set Options.Source or metadata.product.name")
	}

	category := opts.Category
	if category == fex.FindingDetail_CATEGORY_UNSPECIFIED {
		category = fex.FindingDetail_CATEGORY_THREAT
	}

	summary := info.Title
	if summary == "" {
		summary = f.Message
	}
	description := info.Desc
	if description == "" && f.Message != summary {
		description = f.Message
	}

	confidence := confidenceOf(f.ConfidenceID)
	status := statusOf(f.StatusID)

	out := &fex.FindingExchange{
		Id:      info.UID,
		Ref:     refOf(info),
		Summary: summary,
		Details: &fex.FindingDetail{
			Category: category,
			Severity: &fex.Severity{
				Source: &fex.Source{Name: source},
				Rating: severityOf(f.SeverityID),
			},
			Confidence:  confidence,
			Description: description,
			Properties:  findingProperties(f),
		},
		FirstSeenAt: timestamp(first(info.FirstSeenTime, info.CreatedTime, f.Time)),
		LastSeenAt:  timestamp(first(info.LastSeenTime, info.ModifiedTime, f.Time)),
		Status:      status,
		Source:      &fex.Source{Name: source, Url: info.SrcURL},
		Affects:     opts.Affects,
	}
	if out.Affects == nil {
		out.Affects = affectsOf(f.Device)
	}
	if status != fex.Status_STATUS_AFFECTED {
		out.RemediatedAt = timestamp(first(info.ModifiedTime, f.Time))
	}

	tactic, technique := firstAttack(info.Attacks)
	for i, entry := range f.Evidences {
		out.Evidences = append(out.Evidences, Evidences(entry, "evidences."+strconv.Itoa(i), tactic, technique, confidence)...)
	}
	return out, nil
}

// refOf groups findings of the same detection.
func refOf(info ocsf.FindingInfo) string {
	if info.Analytic != nil && info.Analytic.UID != "" {
		return info.Analytic.UID
	}
	return info.Title
}

func findingProperties(f *ocsf.DetectionFinding) map[string]string {
	props := map[string]string{}
	set := func(k, v string) {
		if v != "" {
			props[k] = v
		}
	}

	info := f.FindingInfo
	if a := info.Analytic; a != nil {
		set(PropertyPrefix+"finding_info.analytic.uid", a.UID)
		set(PropertyPrefix+"finding_info.analytic.name", a.Name)
	}
	set(PropertyPrefix+"finding_info.types", strings.Join(info.Types, ","))
	set(PropertyPrefix+"metadata.product.vendor_name", f.Metadata.Product.VendorName)
	set(PropertyPrefix+"metadata.product.version", f.Metadata.Product.Version)
	set(PropertyPrefix+"metadata.version", f.Metadata.Version)

	var tactics, techniques []string
	for _, a := range info.Attacks {
		if a.Tactic != nil && a.Tactic.UID != "" {
			tactics = appendUnique(tactics, a.Tactic.UID)
		}
		if id := techniqueID(a); id != "" {
			techniques = appendUnique(techniques, id)
		}
	}
	set(PropertyTactics, strings.Join(tactics, ","))
	set(PropertyTechniques, strings.Join(techniques, ","))

	for k, v := range f.Unmapped {
		set(PropertyPrefix+"unmapped."+k, v)
	}
	if len(props) == 0 {
		return nil
	}
	return props
}

// affectsOf names the device as the affected component. The device uid is the
// component id when there is one, and the hostname is the identifier the
// platform matches assets on.
func affectsOf(d *ocsf.Device) []*fex.Affects {
	if d == nil {
		return nil
	}
	id := d.UID
	if id == "" {
		id = d.Hostname
	}
	if id == "" {
		id = d.Name
	}
	if id == "" {
		return nil
	}

	identifiers := map[string]string{}
	if d.Hostname != "" {
		identifiers["hostname"] = d.Hostname
	}
	if d.UID != "" {
		identifiers[PropertyPrefix+"device.uid"] = d.UID
	}
	if os := d.OS; os != nil {
		if os.Name != "" {
			identifiers["os-platform"] = os.Name
		}
		if os.Version != "" {
			identifiers["os-version"] = os.Version
		}
	}
	return []*fex.Affects{{Component: &fex.Component{Id: id, Identifiers: identifiers}}}
}

func statusOf(id int) fex.Status {
	switch id {
	case ocsf.StatusResolved, statusArchived, statusDeleted:
		return fex.Status_STATUS_FIXED
	case ocsf.StatusSuppressed:
		return fex.Status_STATUS_NOT_AFFECTED
	default:
		// New, In Progress, Unknown and Other: a detection that is reported and
		// not closed is open.
		return fex.Status_STATUS_AFFECTED
	}
}

func severityOf(id int) fex.SeverityRating {
	switch id {
	case ocsf.SeverityCritical, ocsf.SeverityFatal:
		return fex.SeverityRating_SEVERITY_RATING_CRITICAL
	case ocsf.SeverityHigh:
		return fex.SeverityRating_SEVERITY_RATING_HIGH
	case ocsf.SeverityMedium:
		return fex.SeverityRating_SEVERITY_RATING_MEDIUM
	case ocsf.SeverityLow:
		return fex.SeverityRating_SEVERITY_RATING_LOW
	case ocsf.SeverityInformational:
		return fex.SeverityRating_SEVERITY_RATING_NONE
	default:
		return fex.SeverityRating_SEVERITY_RATING_UNSPECIFIED
	}
}

func confidenceOf(id int) fex.Confidence {
	switch id {
	case ocsf.ConfidenceHigh:
		return fex.Confidence_CONFIDENCE_HIGH
	case ocsf.ConfidenceMedium:
		return fex.Confidence_CONFIDENCE_MEDIUM
	case ocsf.ConfidenceLow:
		return fex.Confidence_CONFIDENCE_LOW
	default:
		return fex.Confidence_CONFIDENCE_UNSPECIFIED
	}
}

// firstAttack is the tactic and technique an evidence carries: those of the
// first attack that names them.
func firstAttack(attacks []ocsf.Attack) (*fex.AttackTactic, *fex.AttackTechnique) {
	var tactic *fex.AttackTactic
	var technique *fex.AttackTechnique
	for _, a := range attacks {
		if tactic == nil && a.Tactic != nil && a.Tactic.UID != "" {
			tactic = &fex.AttackTactic{Id: a.Tactic.UID, Name: a.Tactic.Name}
		}
		if technique == nil {
			if id := techniqueID(a); id != "" {
				technique = &fex.AttackTechnique{Id: id, Name: techniqueName(a)}
			}
		}
	}
	return tactic, technique
}

// techniqueID prefers the sub-technique: T1059.004 is more specific than T1059,
// and ATT&CK sub-technique ids are technique ids.
func techniqueID(a ocsf.Attack) string {
	if a.SubTechnique != nil && a.SubTechnique.UID != "" {
		return a.SubTechnique.UID
	}
	if a.Technique != nil {
		return a.Technique.UID
	}
	return ""
}

func techniqueName(a ocsf.Attack) string {
	if a.SubTechnique != nil && a.SubTechnique.UID != "" {
		return a.SubTechnique.Name
	}
	if a.Technique != nil {
		return a.Technique.Name
	}
	return ""
}

// timestamp converts an OCSF timestamp, milliseconds since the epoch, and
// leaves an unset one nil.
func timestamp(ms int64) *timestamppb.Timestamp {
	if ms <= 0 {
		return nil
	}
	return timestamppb.New(time.UnixMilli(ms).UTC())
}

func first(values ...int64) int64 {
	for _, v := range values {
		if v > 0 {
			return v
		}
	}
	return 0
}

func appendUnique(list []string, v string) []string {
	for _, have := range list {
		if have == v {
			return list
		}
	}
	return append(list, v)
}
