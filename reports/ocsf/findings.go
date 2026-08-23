// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package ocsf

import (
	"strings"

	"github.com/cockroachdb/errors"
)

// FindingClasses selects which OCSF class a check result is reported as.
//
// Every check is reported exactly once, in one class. That is what other OCSF
// producers do: Security Lake routes each Security Hub finding to the one class
// that fits it, and Prowler reports every check as a Detection Finding. Emitting
// a check as two events would double it in anything that counts findings across
// classes.
//
// A cnspec check is a compliance check, so Compliance Finding (2003) is the
// default: it has a compliance object for the framework mappings and the control.
// Detection Finding (2004) is what Splunk Enterprise Security and similar tools
// model findings on; it has no compliance object, so the mappings travel in
// unmapped, and it has the risk and impact attributes 2003 lacks.
type FindingClasses byte

const (
	FindingsCompliance FindingClasses = iota + 1
	FindingsDetection
)

// findingClassNames is the vocabulary ParseFindingClasses accepts, in the order
// it lists them back to a user who typed something else.
var findingClassNames = []struct {
	name    string
	classes FindingClasses
}{
	{"compliance", FindingsCompliance},
	{"detection", FindingsDetection},
}

// ParseFindingClasses resolves a user-provided finding class. Like ParseVersion
// the set is closed: an unrecognized value would otherwise silently fall back to
// the default and produce a stream of the class the user did not ask for.
func ParseFindingClasses(raw string) (FindingClasses, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return FindingsCompliance, nil
	}
	for _, known := range findingClassNames {
		if known.name == value {
			return known.classes, nil
		}
	}
	return 0, errors.Newf("unknown OCSF finding class %q, supported classes are: %s",
		raw, strings.Join(SupportedFindingClasses(), ", "))
}

// SupportedFindingClasses lists the finding classes cnspec can report checks as.
func SupportedFindingClasses() []string {
	res := make([]string, len(findingClassNames))
	for i := range findingClassNames {
		res[i] = findingClassNames[i].name
	}
	return res
}
