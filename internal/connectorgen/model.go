// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package connectorgen extracts connector metadata from mql provider source.
//
// The interactive launcher builds an input form per connector. Three facts
// decide what that form looks like, and only one of them is available at
// runtime:
//
//   - What the connector declares -- flags, argument counts, discovery targets.
//     A provider ships this as <name>.json, a verbatim json.Marshal of
//     plugin.Provider, so cnspec can already read it from an installed
//     provider.
//   - Which environment variable backs which flag. This exists only as Go
//     control flow inside the provider's own ParseCLI.
//   - The sub-command vocabulary -- `github org <ORG>` versus `github repo
//     <ORG>/<REPO>`. This exists only as `case "org":` in the same function and
//     as prose in the connector's Long text.
//
// The last two were previously established by reading provider source by hand
// and running provider binaries, then hand-copying the result into cnspec. This
// package reads them mechanically instead.
//
// It parses source with go/ast and nothing else: no go/types, no module
// resolution, no compilation. There are 81 provider modules in the mql monorepo
// and a dozen more in a second repository pinned to an older SDK, each with its
// own go.mod, so loading them is not viable. Everything below is therefore a
// syntactic claim about the source, and everything the syntax does not settle
// is recorded as a Gap rather than guessed at. A plausible-looking wrong
// credential route drops the secret in silence; an honest "could not determine"
// does not.
package connectorgen

// SchemaVersion is the artifact format version. Bump it when a consumer would
// have to change, not for an added optional field.
const SchemaVersion = 1

// Artifact is the whole generated file: what was read, what came out, and what
// could not be determined.
type Artifact struct {
	Schema int `json:"schema"`
	// GeneratedBy names the tool, so someone finding the file knows not to edit
	// it by hand.
	GeneratedBy string `json:"generated_by"`
	// Sources records the repositories the facts came from. It carries the
	// commit rather than the path: a checked-in artifact whose diff depends on
	// whose laptop ran the generator is not reviewable.
	Sources []Source `json:"sources"`
	// Connectors is the extraction, sorted by name then provider.
	Connectors []Connector `json:"connectors"`
	// Gaps is what the walk could not determine, and is the point of the
	// exercise: it is the specification for what the provider SDK would have to
	// carry for cnspec to stop reading Go source at all.
	Gaps []Gap `json:"gaps"`
}

// Source is one repository the extraction read.
type Source struct {
	// Name is the repository's short name, e.g. "mql".
	Name string `json:"name"`
	// Commit is the checked-out revision, or empty when the tree is not a git
	// repository.
	Commit string `json:"commit,omitempty"`
	// Dirty reports that the tree had uncommitted changes, which makes Commit
	// an incomplete description of what was read.
	Dirty bool `json:"dirty,omitempty"`
	// Providers is how many provider declarations were found in this tree.
	Providers int `json:"providers"`
}

// Connector is one selectable target: a (provider, connector) pair. The first
// block of fields mirrors plugin.Connector and is also obtainable from an
// installed provider; the second block is what only the source carries.
type Connector struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Use      string `json:"use,omitempty"`
	Short    string `json:"short,omitempty"`
	// Long is the connector's full help text. Several providers document their
	// sub-command shape nowhere else.
	Long     string   `json:"long,omitempty"`
	Aliases  []string `json:"aliases,omitempty"`
	MinArgs  uint     `json:"min_args,omitempty"`
	MaxArgs  uint     `json:"max_args,omitempty"`
	IsHidden bool     `json:"is_hidden,omitempty"`
	Maturity string   `json:"maturity,omitempty"`
	// Flags are sorted by Long, so a regeneration diff shows what changed
	// rather than how the author reordered the literal.
	Flags     []Flag   `json:"flags,omitempty"`
	Discovery []string `json:"discovery,omitempty"`

	// Env names the environment variables the provider reads a flag's value
	// from, in the order ParseCLI consults them.
	Env []FlagEnv `json:"env,omitempty"`
	// Positional is the sub-command vocabulary: the literal words the provider
	// compares a positional argument against.
	Positional []Positional `json:"positional,omitempty"`
	// CarriedForward marks a connector this extraction did not produce, kept
	// from the previous artifact so a regeneration never loses one. Its
	// metadata is as old as whenever it was last derived.
	CarriedForward bool `json:"carried_forward,omitempty"`
	// Gaps are this connector's share of Artifact.Gaps, repeated here so a
	// consumer reading one connector sees what is missing from it.
	Gaps []Gap `json:"gaps,omitempty"`
}

// Flag mirrors plugin.Flag. Type and Option are the numeric enum values, the
// same ones the runtime metadata carries, because that is what a consumer
// compares against plugin.FlagType_String.
type Flag struct {
	Long    string `json:"long"`
	Short   string `json:"short,omitempty"`
	Default string `json:"default,omitempty"`
	Type    int32  `json:"type,omitempty"`
	Option  int32  `json:"option,omitempty"`
	Desc    string `json:"desc,omitempty"`
	// Config is the flag's ConfigEntry. It decides whether a value can be
	// delivered through MONDOO_<FLAG> at all: mql reads that variable only for
	// a flag declaring no config mapping, and "-" means the flag comes from
	// cobra alone.
	Config string `json:"config,omitempty"`
}

// FlagEnv is one flag and the environment variables that back it.
type FlagEnv struct {
	// Flag is the flag's Long name. It is not necessarily declared by this
	// connector -- see Declared.
	Flag string `json:"flag"`
	// Vars are the variables consulted, in the order the source consults them.
	// More than one is a fallback chain, and the first is what the provider
	// prefers.
	Vars []string `json:"vars"`
	// Func is the function the association was read out of.
	Func string `json:"func"`
	// Via says where the provider reads the variable: "parse-cli" when it is
	// read while parsing the command line, "connection" when the connection
	// package reads it at connect time from the environment directly.
	//
	// The distinction is not cosmetic. A "parse-cli" variable is consulted only
	// when the flag is absent, so a launcher may set either. A "connection"
	// variable is read by the process that connects, and several of them take
	// precedence over the flag rather than yielding to it, so a launcher that
	// sets one is overriding whatever the user typed.
	Via string `json:"via,omitempty"`
	// Declared reports whether this connector declares a flag by that name. A
	// false here is a real finding rather than noise: it means the provider
	// reads a value the CLI has no way to pass.
	Declared bool `json:"declared"`
	// Composed marks an association where the variable is built from more than
	// one environment variable rather than chosen from them -- okta joins
	// OKTA_ORG_NAME and OKTA_BASE_URL with a "." -- so Vars is not a fallback
	// list and setting only the first of them will not work.
	Composed bool `json:"composed,omitempty"`
}

// Positional is the vocabulary for one positional argument.
type Positional struct {
	// Index is the argument position, counted the way the provider counts it:
	// req.Args[0] is index 0.
	Index int `json:"index"`
	// Values are the literal words compared against, in source order.
	Values []string `json:"values"`
	// Func is the function the comparison was read out of.
	Func string `json:"func"`
}

// Gap is one thing the walk could not determine, or determined only partly.
//
// Kind is a stable machine-readable label so the report can be grouped and so
// one run can be diffed against the next; Detail is the human sentence. Both
// are needed: the label alone does not say which flag, and the sentence alone
// cannot be counted.
type Gap struct {
	Provider  string `json:"provider,omitempty"`
	Connector string `json:"connector,omitempty"`
	Kind      string `json:"kind"`
	Detail    string `json:"detail"`
	// Where is the source position, relative to the repository root, so the gap
	// can be opened.
	Where string `json:"where,omitempty"`
}

// The gap kinds. Each names a distinct reason the source did not settle a
// question, because "could not determine" collected into one bucket cannot be
// turned into a list of SDK changes.
const (
	// GapNoProviderPkg: the provider declares connectors but ships no provider/
	// package next to its config, so there is no ParseCLI to read.
	GapNoProviderPkg = "no-provider-package"
	// GapNoParseCLI: the provider package has no ParseCLI function.
	GapNoParseCLI = "no-parsecli"
	// GapCarriedForward: a connector the previous artifact had and this
	// extraction did not produce, kept rather than dropped.
	GapCarriedForward = "carried-forward"
	// GapSourceAbsent: a provider directory is in the tree but its source is
	// not -- only a built artifact or a resource schema. Nothing about that
	// connector can be read here, and it will simply be missing.
	GapSourceAbsent = "provider-source-absent"
	// GapDynamicGetenv: os.Getenv was called with something other than a
	// literal or a resolvable constant, so the variable name is not knowable
	// from the syntax.
	GapDynamicGetenv = "dynamic-getenv"
	// GapEnvOutsideParseCLI: an environment variable is read in the provider
	// package but in a function neither ParseCLI nor Connect reaches, so
	// nothing says which flag it belongs to.
	GapEnvOutsideParseCLI = "env-outside-parsecli"
	// GapUnboundEnv: an environment variable is read inside ParseCLI but its
	// value never meets a flag, so it backs no flag this tool can name.
	GapUnboundEnv = "unbound-env"
	// GapAmbiguousBinding: a variable carries more than one flag name, so
	// pairing it with an environment variable would be a guess.
	GapAmbiguousBinding = "ambiguous-binding"
	// GapAlternativeBranches: the flag and the environment variable are the two
	// arms of one if/else chain and never meet in a single value, so only the
	// author's intent -- not the syntax -- says they are the same credential.
	GapAlternativeBranches = "alternative-branches"
	// GapUndeclaredFlag: the provider reads a flag name no connector declares.
	GapUndeclaredFlag = "undeclared-flag"
	// GapConfigNotLiteral: some part of the plugin.Provider literal is not a
	// composite literal -- a function call, or a value assembled elsewhere.
	GapConfigNotLiteral = "config-not-literal"
	// GapUnresolvedConstant: an identifier used as a flag name, a discovery
	// target or a literal value did not resolve to a string in the source that
	// was read.
	GapUnresolvedConstant = "unresolved-constant"
	// GapNoPositionalVocabulary: the connector takes positional arguments but
	// no literal comparison against them was found, so the words it accepts are
	// not knowable from the syntax.
	GapNoPositionalVocabulary = "no-positional-vocabulary"
	// GapComputedPositional: a positional argument is compared against
	// something other than a literal -- a slice, a map lookup, a regexp -- so
	// the vocabulary is computed rather than written down.
	GapComputedPositional = "computed-positional"
	// GapPositionalAmbiguous: the provider ships several connectors and one
	// ParseCLI, so a vocabulary found there cannot be attributed to one of
	// them.
	GapPositionalAmbiguous = "positional-ambiguous"
)
