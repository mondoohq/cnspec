// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cnspec

import (
	"regexp"
	"strings"

	"go.mondoo.com/mql/cli/config"
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

// defaultUpdatesURL is the install service cnspec resolves updates through when
// nothing is configured. `updates_url` points at the same kind of service, which
// is also where providers resolve from, at `updates_url + "/providers"`.
const defaultUpdatesURL = "https://install.mondoo.com"

// ReleaseURL returns the release manifest the binary self-update reads. It lives
// here so the implicit update in main and the explicit `cnspec update` command
// cannot drift onto different releases.
//
// The channel travels as a query parameter rather than a different document
// name. The install service's routes are named after the package, not after the
// manifest -- there is no /package/cnspec/preview.json -- and an unmatched path
// there is answered with the landing page as a cacheable 200, so a path-shaped
// channel would hand the updater HTML with a success status. An empty or stable
// channel produces exactly the URL this returned before, so nothing changes for
// a client that never sets one.
func ReleaseURL(updatesURL string, channel string) string {
	if updatesURL == "" {
		updatesURL = defaultUpdatesURL
	}

	url := strings.TrimSuffix(updatesURL, "/") + "/package/cnspec/latest.json"
	if channel != "" && channel != config.ChannelStable {
		url += "?channel=" + channel
	}
	return url
}

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
