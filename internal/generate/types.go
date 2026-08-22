// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generate

// Check is the provider-neutral view of a policy query the generator works on.
// The command layer converts bundle queries into Checks and writes generated MQL
// back, so this package never depends on the YAML/bundle types directly.
type Check struct {
	// UID is the query's unique identifier, used for reporting and as a weak
	// provider hint.
	UID string
	// Title and Desc are the natural-language intent, the source of generation.
	Title string
	Desc  string
	// Filters holds the query's filter expressions (asset.platform == "...").
	// They are the authoritative signal for the target provider.
	Filters []string
	// Mql is the existing query, empty when generation is requested.
	Mql string
	// Props are the parameterized properties available to the check, surfaced to
	// the agent so it references them (props.<name>) rather than inventing
	// literals.
	Props []Prop
	// HasVariants marks a variant-parent check whose MQL lives in its per-platform
	// variants. Such a check must not get MQL generated onto the parent.
	HasVariants bool
	// Guidance is optional extra author steering (e.g. feedback from an
	// interactive "regenerate" step), added to the prompt.
	Guidance string
}

// Prop is a parameterized property a check can reference in its MQL.
type Prop struct {
	Name string
	Type string
	Desc string
}

// Action is the outcome of processing one check.
type Action string

const (
	// ActionGenerated means new MQL was produced and validated.
	ActionGenerated Action = "generated"
	// ActionSkipped means the check already had MQL (and --force was not set) or
	// lacked the intent needed to generate.
	ActionSkipped Action = "skipped"
	// ActionFailed means generation or validation failed for this check.
	ActionFailed Action = "failed"
)

// Result is the per-check outcome returned to the caller.
type Result struct {
	UID         string
	Action      Action
	Provider    string
	MQL         string
	Explanation string
	// Reason explains a skip or failure in one line.
	Reason string
	Err    error
}
