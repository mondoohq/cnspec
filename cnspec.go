// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cnspec

import (
	"regexp"
)

// Version is set via ldflags
var Version string

// Build version is set via ldflags
var Build string

// Date is set via ldflags
var Date string

/*
 versioning follows semver guidelines: https://semver.org/

<valid semver> ::= <version core>
                 | <version core> "-" <pre-release>
                 | <version core> "+" <build>
                 | <version core> "-" <pre-release> "+" <build>

<version core> ::= <major> "." <minor> "." <patch>

<major> ::= <numeric identifier>

<minor> ::= <numeric identifier>

<patch> ::= <numeric identifier>
*/

// GetVersion returns the version of the build
// valid semver version including build version (e.g. 4.10.0+4900), where 4900 is a forward rolling int
func GetVersion() string {
	if Version == "" {
		return "unstable"
	}
	return Version
}

var coreSemverRegex = regexp.MustCompile(`^(\d+.\d+.\d+)`)

// GetCoreVersion returns the semver core (i.e. major.minor.patch)
func GetCoreVersion() string {
	v := Version

	if v != "" {
		v = coreSemverRegex.FindString(v)
	}

	if v == "" {
		return "unstable"
	}
	return v
}

// GetBuild returns the git sha of the build
func GetBuild() string {
	b := Build
	if len(b) == 0 {
		b = "development"
	}
	return b
}

// GetDate returns the date of this build
func GetDate() string {
	d := Date
	if len(d) == 0 {
		d = "unknown"
	}
	return d
}

var majorVersionRegex = regexp.MustCompile(`^(\d+)`)

// APIVersion is the major version of the version string (e.g. 4)
func APIVersion() string {
	v := Version

	if v != "" {
		v = majorVersionRegex.FindString(v)
	}

	if v == "" {
		return "unstable"
	}
	return v
}

// Info on this application with version and build
func Info() string {
	return "cnspec " + GetVersion() + " (" + GetBuild() + ", " + GetDate() + ")"
}
