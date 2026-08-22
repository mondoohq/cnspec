// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package forms

import (
	"path/filepath"
	"runtime"
)

// FormSpec describes a connector's input screen: which fields appear, in what
// order, which are the target and which are the credential, and where a field's
// suggested values come from.
//
// This is deliberately a data shape rather than Go logic, because it is the
// shape proposed for the provider SDK. A provider that declares its own form
// emits exactly this, and providerFormSpec, over in the launcher, is the single
// place that reading it plugs in. The specs registered here today are the
// interim source for the connectors people reach for most; each one deletes
// itself the day its provider ships the same declaration.
type FormSpec struct {
	// Positional replaces the derived positional fields wholesale, because the
	// usage string cannot describe a sub-command shape like `github org <ORG>`.
	Positional []PositionalSpec `json:"positional,omitempty"`
	// Target and Credential move flags into those sections, in this order.
	Target     []string `json:"target,omitempty"`
	Credential []string `json:"credential,omitempty"`
	// Secret and NotSecret override the shared classifier for one flag of one
	// connector.
	//
	// The classifier in secrets_classify.go reads flag names for every
	// connector at once, so correcting one connector's odd flag by widening a
	// word list re-classifies every other flag that happens to contain that
	// word. Saying it here keeps the correction where the person who found it
	// is already working, and keeps it from reaching anyone else's connector.
	//
	// Secret is the safe direction and NotSecret is not: marking a real
	// credential NotSecret puts it on the command line, where `ps auxww` reads
	// it. Use it only for a flag that names where a credential lives -- a path,
	// an id -- and that the provider therefore has to receive as an argument.
	Secret    []string `json:"secret,omitempty"`
	NotSecret []string `json:"not_secret,omitempty"`
	// Labels renames a flag for display; Sources attaches a value picker;
	// Choices attaches a static option list.
	Labels  map[string]string `json:"labels,omitempty"`
	Sources map[string]string `json:"sources,omitempty"`
	// LiveSources attach a picker that only runs when the field is opened,
	// merged into whatever Sources already found.
	LiveSources map[string]string   `json:"live_sources,omitempty"`
	Choices     map[string][]string `json:"choices,omitempty"`
	// Hide drops flags that are noise in a launcher.
	Hide []string `json:"hide,omitempty"`
	// ShowFlagsIf hides a flag unless the first positional field holds one of
	// the listed values, so a form only asks what the chosen shape needs.
	ShowFlagsIf map[string][]string `json:"show_flags_if,omitempty"`
	// Env names the environment variable a field's value travels in, keyed by
	// field identity -- "f:profile", "p:1", or a bare flag name as shorthand.
	//
	// Not every value the user picks has a flag to travel in. alicloud
	// declares no --profile, snowflake takes no connection name, docker and
	// container take no --context, and k8s --context is parsed and then never
	// reaches the client config. For those the child process is reached
	// through its environment instead, and this is where a spec says so. The
	// general case -- a value that has to be written to a file before it means
	// anything, which is what k8s needs -- is an EnvSpec instead.
	Env map[string]string `json:"env,omitempty"`
}

// PositionalSpec describes one field in the TARGET section: a positional
// argument, or -- with Special set -- a question the launcher owns that
// contributes no argument at all.
type PositionalSpec struct {
	Label    string `json:"label"`
	Desc     string `json:"desc,omitempty"`
	Required bool   `json:"required,omitempty"`
	// Special marks this field as the launcher's rather than the connector's,
	// so it is drawn and answered like a target but never reaches the command
	// line -- not as an argument, and not as a flag.
	//
	// This is what makes a value with no flag expressible. alicloud has a
	// profile and no --profile; docker, container and local have a context and
	// no --context. Both have to travel in the child's environment, and until
	// this existed there was no way to say so: the only field a spec could add
	// for a flagless value was a positional, args() emits every visible
	// positional, and alicloud is MinArgs=0/MaxArgs=0 -- so `cnspec shell
	// alicloud staging` answered `unknown command "staging"`. The obvious
	// escape, an Emit map with no entries, suppressed the argument by making
	// emitted() return "" for every value, but formEnvironment reads the same
	// accessor, so the child got ALIBABA_CLOUD_PROFILE= instead. An empty
	// variable is not an absent one.
	//
	// The name is the field's identity as "s:<Special>", which is how Env and
	// a source's Needs refer to it, and it has to be declared in
	// launcherOwnedFields -- see TestNoKeychainToggleIsOffered for why that
	// allowlist exists.
	Special string `json:"special,omitempty"`
	// Options makes it a picker; Source names a live value picker instead.
	Options []string `json:"options,omitempty"`
	Source  string   `json:"source,omitempty"`
	// Emit maps a chosen option to what it contributes to the command line. An
	// option mapping to "" contributes nothing, which is how a selector steers
	// the UI without adding a word cnspec would not understand.
	Emit map[string]string `json:"emit,omitempty"`
	// SourceBy picks this field's value source from the value of the preceding
	// positional argument.
	SourceBy map[string]string `json:"source_by,omitempty"`
	// LiveSourceBy attaches a picker that only runs when the field is opened,
	// chosen by the preceding positional the same way SourceBy is.
	LiveSourceBy map[string]string `json:"live_source_by,omitempty"`
	// DiscoverBy preselects --discover targets for a chosen value, for targets
	// that are enumerated by cnspec's own discovery rather than locally.
	DiscoverBy map[string][]string `json:"discover_by,omitempty"`
	// ShowIf hides this field unless the preceding positional holds one of
	// these values.
	ShowIf []string `json:"show_if,omitempty"`
}

// The launcher-owned target fields, named here because two things refer to
// each one: the spec that creates it, and whatever names its identity --
// "s:alicloud-profile" -- in an Env map or a source's Needs. A literal in both
// places is a typo waiting to be silent, since fieldMatchesIdentity simply
// never matches and the value goes nowhere.
//
// Every one of these must also be listed in launcherOwnedFields; see
// TestNoKeychainToggleIsOffered.
const (
	// SpecialAlicloudProfile is a profile from ~/.alibabacloud/credentials.
	// alicloud declares no --profile, so it travels as ALIBABA_CLOUD_PROFILE.
	SpecialAlicloudProfile = "alicloud-profile"
	// SpecialDockerContext is a docker context. docker and container declare
	// no --context, so it travels as DOCKER_CONTEXT.
	SpecialDockerContext = "docker-context"
)

// formSpecs holds every registered overlay, keyed by connector name.
//
// It used to be one map literal, which meant every connector curated after the
// first twelve was an edit to the same block of the same file. It is a registry
// for the same reason the source registry is one: adding a connector should be
// an append in a file its author owns.
var formSpecs = map[string]FormSpec{}

// uncurated records the connectors that deliberately have no FormSpec, and
// why. It is the other half of formSpecs: without it, "nobody has curated this
// yet" and "there is nothing a spec could say about this" are the same empty
// entry, and the reach test cannot tell a gap from a decision.
//
// The reasons are load-bearing rather than decorative. Each one is a claim
// about the connector that stops being true if the connector changes -- device
// hiding every flag it declares, ipinfo declaring none at all -- so a reason
// that has gone stale is a spec that should now be written.
var uncurated = map[string]string{}

// registerUncurated declares that a connector has no form on purpose, and is
// called from the init of the file that owns it, beside the specs its
// neighbours got.
func registerUncurated(connector, reason string) {
	uncurated[connector] = reason
	specFiles[connector] = callerFile()
}

// uncuratedReason returns the recorded reason a connector has no spec.
func uncuratedReason(connector string) (string, bool) {
	reason, ok := uncurated[connector]
	return reason, ok
}

// duplicateSpecs records connectors registered more than once, for
// TestNoConnectorHasTwoSpecs to report. Two registrations mean two people each
// believed they owned the connector, and whichever init ran second would
// otherwise silently win -- a merge conflict that leaves no conflict marker.
var duplicateSpecs []string

// registerSpec declares a connector's form, and is called from each spec file's
// init. The first registration wins; a later one is recorded and ignored rather
// than allowed to overwrite, because init order across files is not something
// either author controls.
func registerSpec(connector string, spec FormSpec) {
	if _, taken := formSpecs[connector]; taken {
		duplicateSpecs = append(duplicateSpecs, connector)
		return
	}
	formSpecs[connector] = spec
	specFiles[connector] = callerFile()
}

// specFiles records which file each registration came from, so
// TestEverySpecIsFiledUnderItsCategory can check that a spec is written where
// the launcher shows its connector.
//
// It is taken from the call stack rather than declared, because a declared file
// name is a second thing to keep in step and would be wrong in exactly the case
// the test exists to catch: a spec pasted into the wrong file, along with the
// name of the file it came from. The stack cannot be pasted.
var specFiles = map[string]string{}

// callerFile is the base name of the spec file that called register*. Two
// frames up: this function, then the register* that called it.
func callerFile() string {
	if _, file, _, ok := runtime.Caller(2); ok {
		return filepath.Base(file)
	}
	return ""
}

// Specs returns every registered overlay, keyed by connector name.
//
// The map is a copy: the registry is written once, by the inits in this
// package, and a caller that could reach the live map could add a connector's
// form from anywhere -- which is the arrangement registerSpec's first-wins rule
// exists to prevent.
func Specs() map[string]FormSpec {
	out := make(map[string]FormSpec, len(formSpecs))
	for k, v := range formSpecs {
		out[k] = v
	}
	return out
}

// SpecFor returns the curated overlay for a connector, if one is registered.
func SpecFor(connector string) (FormSpec, bool) {
	spec, ok := formSpecs[connector]
	return spec, ok
}

// Uncurated returns every connector deliberately left without a form, and the
// reason recorded for it.
func Uncurated() map[string]string {
	out := make(map[string]string, len(uncurated))
	for k, v := range uncurated {
		out[k] = v
	}
	return out
}

// UncuratedReason returns the recorded reason a connector has no spec.
func UncuratedReason(connector string) (string, bool) {
	return uncuratedReason(connector)
}

// Duplicates returns the connectors registered more than once. Two
// registrations mean two people each believed they owned the connector; see
// registerSpec.
func Duplicates() []string {
	return append([]string(nil), duplicateSpecs...)
}
