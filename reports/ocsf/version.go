// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package ocsf

import (
	"slices"
	"strings"

	"github.com/cockroachdb/errors"
)

// Version identifies the OCSF schema version that an event set is emitted for.
// It ends up in `metadata.version` of every event.
type Version string

const (
	// Version130 is the highest OCSF version Amazon Security Lake accepts for
	// custom sources, which makes it the safe default: it is ingestible by the
	// lake and readable by every other OCSF consumer.
	Version130 Version = "1.3.0"

	// Version190 is the current OCSF release. Pick it for SIEMs that track the
	// latest schema; Security Lake rejects it today.
	Version190 Version = "1.9.0"

	// DefaultVersion is what cnspec emits unless the user asks for another one.
	DefaultVersion = Version130
)

// versions is ordered oldest to newest; the order is what AtLeast compares on.
var versions = []Version{Version130, Version190}

// ParseVersion resolves a user-provided version string. The set is closed on
// purpose: an unrecognized version would otherwise be written into
// `metadata.version` while the events keep the shape of another version.
func ParseVersion(raw string) (Version, error) {
	v := Version(strings.TrimSpace(raw))
	if v == "" {
		return DefaultVersion, nil
	}
	if slices.Contains(versions, v) {
		return v, nil
	}
	return "", errors.Newf("unsupported OCSF version %q, supported versions are: %s",
		raw, strings.Join(SupportedVersions(), ", "))
}

// AtLeast reports whether v is other or newer.
func (v Version) AtLeast(other Version) bool {
	return slices.Index(versions, v) >= slices.Index(versions, other)
}

func (v Version) String() string { return string(v) }

// SupportedVersions lists the OCSF versions cnspec can emit, oldest first.
func SupportedVersions() []string {
	res := make([]string, len(versions))
	for i := range versions {
		res[i] = string(versions[i])
	}
	return res
}
