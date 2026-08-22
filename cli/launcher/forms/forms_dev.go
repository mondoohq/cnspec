// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package forms

// Developer & Supply Chain: artifactory, depsdev and sbom.
//
// Two of the three are path-shaped -- sbom reads a bill of materials off disk,
// depsdev takes a package coordinate -- so the useful form is the one that
// names the argument in the connector's own vocabulary rather than repeating
// the usage string. artifactory is the odd one out and is deliberately left
// uncurated; the reason is in the init below, in code, because "nobody got to
// it" and "there is nothing a spec can say" are indistinguishable from an empty
// registry.

func init() {
	// artifactory reaches the catalog from providers.DefaultProviders, the
	// static list compiled into the binary, which carries no Flags, no MinArgs
	// and no MaxArgs. On a machine that has not installed the provider,
	// HasFormData() is false and the launcher shows its free-text argument box
	// -- a spec would never be applied at all. On a machine that has installed
	// it the flags are real (--url, --token, --api-key, verified against
	// 13.1.0), but internal/connectors/connectors.json has no artifactory entry, so
	// TestEverySpecNamesRealFlags logs "no snapshot entry" and checks nothing.
	// A spec would ship on one laptop's authority, which is exactly the
	// condition the snapshot was introduced to end.
	//
	// Refreshing the snapshot does not fix it: the provider is not installed on
	// the machine the snapshot was last taken on either, and `-update` records
	// what BuildCatalog found rather than what it wished it had found. db2 is
	// in the same position, and so are auth0, bitwarden, dropbox, jumpcloud,
	// keycloak and zoom -- all of which TestEverySpecNamesRealFlags reports by
	// name.
	//
	// Its credential is registered below regardless: that is a fact about the
	// provider, not a claim about a form, and without it the generic form
	// refuses to launch.
	registerUncurated("artifactory",
		"static-list only, so it carries no flags and has no snapshot entry to check a spec against")

	// depsdev is the one connector here whose target is genuinely optional:
	// with a go.mod it reports on that module's direct dependencies, without
	// one it answers questions about individual packages by name. So the
	// positional is not Required, and the description says what leaving it
	// empty means rather than leaving the reader to find out by launching.
	//
	// --path and the positional were both run against the installed provider
	// and parse the same file, so the flag is hidden rather than offered as a
	// second way to say the same thing.
	registerSpec("depsdev", FormSpec{
		Positional: []PositionalSpec{{
			Label: "go.mod path",
			Desc:  "the go.mod whose dependencies to check; leave empty to query packages by name",
		}},
		Hide: []string{"path"},
	})

	// sbom reads a CycloneDX or SPDX document off disk. --format is left as
	// free text with only its label improved: the provider is not installed on
	// this machine and its source is not in the module graph, so the strings
	// its --format accepts could only be guessed. mql's own sbom package
	// decodes cyclonedx-json, cyclonedx-xml, spdx-json, spdx-tag-value and
	// json, but whether the provider spells them that way is exactly the kind
	// of plausible-looking assumption that has already cost this branch once.
	// Auto-detect is the default and needs no answer.
	registerSpec("sbom", FormSpec{
		Positional: []PositionalSpec{{
			Label: "SBOM file", Desc: "the CycloneDX or SPDX document to scan",
			Required: true,
		}},
		Labels: map[string]string{"format": "format (default: auto-detect)"},
	})
}
