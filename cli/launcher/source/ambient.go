// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package source

import (
	"os"
	"strings"

	tuiform "go.mondoo.com/cnspec/cli/tui/form"
)

// The ambient class: connectors whose entire credential is one token, taken
// from one environment variable or handed over on the spot.
//
// Seven of them have nothing to enumerate. Treating them as value pickers is
// what produced the empty box with no explanation that started this redesign:
// a picker asks "which one", and there is no "which one" -- there is one token,
// and the only questions worth asking about it are whether it is there and
// where it came from.
//
// So an ambient credential gets two widgets instead of a list:
//
//   - a readout saying which variable supplied the token, or what to export if
//     none did. It is a launcher-owned field (`special`), never a flag, and it
//     holds a variable *name* -- never a token.
//   - the connector's own --token flag, redrawn as a paste box, for the case
//     where the shell the user has already left is not where they want to go.
//
// Neither ever puts a credential on the command line: the readout has nothing
// to put there, and the paste box is secret, so args() skips it and delivery.go
// decides which verified route the value travels by.

// EnvLookup reads one environment variable.
//
// It is a parameter rather than a direct call to os.LookupEnv because these are
// exactly the variables a developer running the tests is most likely to have
// exported. A test that consulted the real environment would pass or fail
// depending on whose machine it ran on, and would be reading that developer's
// actual tokens to do it.
type EnvLookup func(name string) (string, bool)

// ambientEnv is the environment the launcher itself reads. Tests replace it
// through SetAmbientEnv.
var ambientEnv EnvLookup = os.LookupEnv

// SetAmbientEnv replaces the environment the credential readouts are derived
// from, and returns what it replaced so a test can put it back.
//
// It is exported because the tests that need it are the launcher's: a readout
// is only observable on a built form, and the form engine and the connector
// catalog are both on the other side of this package. The variables involved
// are exactly the ones a developer running the suite is most likely to have
// exported, so a test that read the real environment would pass or fail by
// whose machine it ran on -- and would be reading that developer's actual
// tokens to do it.
func SetAmbientEnv(env EnvLookup) EnvLookup {
	prev := ambientEnv
	ambientEnv = env
	return prev
}

// SpecialCredentialState marks the readout as a field the launcher owns rather
// than one the provider declared, which is what keeps it off the command line:
// args() skips a positional with a `special` marker, and this field has no flag
// to be skipped by. A connector with a second ambient credential suffixes the
// marker with the credential's Name, so the two readouts keep distinct
// identities across a rebuild of the form.
const SpecialCredentialState = "credential-state"

// The reasons a readout gives for the value it is showing. They are what stops
// carryOver from carrying a readout onto a rebuilt form: a prefilled value is
// the rebuild's own business, and this one is re-derived from the environment
// every time resolveSources runs.
const (
	AmbientWhyEnv   = "in your environment"
	ambientWhyPaste = "typed here"
)

// AmbientCredential declares a connector whose credential is ambient.
type AmbientCredential struct {
	// Connector is the connector command word this credential belongs to.
	Connector string
	// Name distinguishes a second credential on the same connector from the
	// main one, and is empty for the main one. Only digitalocean has a second.
	Name string
	// Source is the id of the ClassAmbient source registered for this
	// credential, or "" for one the launcher can only report on. The source is
	// what the model's own machinery re-checks the environment through; the
	// readout does not depend on it having run.
	Source string
	// Flag is the connector's own flag carrying the same secret, or "" when
	// the provider declares none. A credential with no flag is report-only:
	// there is nowhere for a pasted value to go, and inventing a route for it
	// is how a credential ends up somewhere the provider does not read.
	Flag string
	// Env are the variables the provider reads, in the order it reads them.
	// The first one holding a value is the one that will be used, which is the
	// one the readout names.
	Env []string
	// All says the provider needs every variable in Env rather than the first
	// one that is set, so a partial set is not a credential.
	All bool
	// Label names the readout row, and must not collide with another field's
	// label on the same form -- args() tracks visibility by label.
	Label string
	// Hint is what the readout says when nothing was found. "Empty" and
	// "nothing here to fill in" look identical otherwise, which is the failure
	// this whole widget exists to end, so it names the variable to export.
	Hint string
	// PasteHint is the placeholder on the paste box.
	PasteHint string
	// Compose fills a non-secret field the provider builds out of the
	// environment rather than reading directly. Only okta needs one; see
	// oktaOrganization for why it cannot simply mirror what the provider does.
	Compose func(f *tuiform.Form, env EnvLookup)
}

// ambientCredentials holds every declared ambient credential, in registration
// order. It is a registry rather than a literal for the same reason the source
// and spec registries are: a connector curated later is an append in the file
// that curates it.
var ambientCredentials []AmbientCredential

// RegisterAmbient declares ambient credentials and, for each that names one,
// the ClassAmbient source behind it. Call it from the init of the file that
// owns the connector.
func RegisterAmbient(creds ...AmbientCredential) {
	for _, a := range creds {
		ambientCredentials = append(ambientCredentials, a)
		if a.Source != "" {
			Register(a.source())
		}
	}
}

// AmbientCredentials returns every declared ambient credential, in
// registration order.
//
// It is the read-only view of the registry, for the launcher tests that hold
// all of them to one property at a time -- that every paste box has a route,
// that no two readouts share a label -- which is a form-level question and so
// cannot be asked from here.
func AmbientCredentials() []AmbientCredential {
	return append([]AmbientCredential(nil), ambientCredentials...)
}

// ambientFor returns the credentials declared for a connector.
func ambientFor(connector string) []AmbientCredential {
	var out []AmbientCredential
	for _, a := range ambientCredentials {
		if a.Connector == connector {
			out = append(out, a)
		}
	}
	return out
}

func init() {
	// The variables below are read out of each provider's own ParseCLI, not
	// inferred from its prose. digitalocean and hetzner document theirs in the
	// flag description as well, and both spellings agree.
	RegisterAmbient(
		AmbientCredential{
			Connector: "github", Source: GitHubToken, Flag: "token",
			Env: []string{"GITHUB_TOKEN"},
		},
		AmbientCredential{
			Connector: "gitlab", Source: GitLabToken, Flag: "token",
			Env: []string{"GITLAB_TOKEN"},
		},
		AmbientCredential{
			Connector: "slack", Source: SlackToken, Flag: "token",
			Env: []string{"SLACK_TOKEN"},
		},
		// The three the delivery registry did not already carry, so each
		// declares its own paste route. Every one was read out of the provider
		// that receives it: CLOUDFLARE_TOKEN in cloudflare's connection.go,
		// and DIGITALOCEAN_TOKEN and HCLOUD_TOKEN in their own providers,
		// which also name them in the --token descriptions. Without the route
		// a pasted token has nowhere verified to go and the launcher refuses
		// to run at all, so the paste box would be a dead end for three of the
		// seven.
		AmbientCredential{
			Connector: "cloudflare", Source: CloudflareToken, Flag: "token",
			Env: []string{"CLOUDFLARE_TOKEN"},
		},
		AmbientCredential{
			Connector: "digitalocean", Source: DigitalOceanToken, Flag: "token",
			Env: []string{"DIGITALOCEAN_TOKEN"},
		},
		AmbientCredential{
			Connector: "hetzner", Source: HetznerToken, Flag: "token",
			Env: []string{"HCLOUD_TOKEN"},
		},
		// okta reads three levels in this order: --token, OKTA_API_TOKEN, then
		// OKTA_TOKEN. The readout names whichever one the provider will
		// actually reach for, which is why Env order is load-bearing.
		AmbientCredential{
			Connector: "okta", Source: OktaToken, Flag: "token",
			Env:     []string{"OKTA_API_TOKEN", "OKTA_TOKEN"},
			Compose: composeOktaOrganization,
		},

		// digitalocean's Spaces credentials break the flag/env symmetry every
		// other ambient connector has: DIGITALOCEAN_SPACES_KEY, _SECRET and
		// _REGION are read from the environment and no flag carries any of
		// them. They are surfaced as a readout and deliberately given no paste
		// box, for two reasons.
		//
		// The first is that there is nowhere for a pasted value to go. Every
		// credential now reaches the provider as a *flag* value, over its own
		// ParseCLI, and these three name no flag: digitalocean's connection
		// reads them straight out of the environment it inherits. A paste box
		// would collect a value with nothing to hand it to.
		//
		// The second is that a readout is the whole of what is missing here.
		// Without these variables `--discover spaces-buckets` finds nothing and
		// says nothing about why, which is the same silent-empty failure the
		// ambient class exists to end. Naming the variables in the row is the
		// fix; carrying their values is not.
		AmbientCredential{
			Connector: "digitalocean", Name: "spaces",
			Env:   []string{"DIGITALOCEAN_SPACES_KEY", "DIGITALOCEAN_SPACES_SECRET"},
			All:   true,
			Label: "spaces keys",
			// Named in full rather than abbreviated to a shared prefix: this
			// is the line a user copies into their shell, and _REGION is left
			// out of it because the provider only insists on the other two.
			Hint: "env only — export DIGITALOCEAN_SPACES_KEY and " +
				"DIGITALOCEAN_SPACES_SECRET to scan buckets",
		},
	)
}

// Special is the launcher-owned marker on this credential's readout field.
//
// It is exported because the launcher keeps the allowlist of markers a form is
// allowed to carry, and a marker that is not on it is how a launcher-owned
// field would quietly become a positional argument.
func (a AmbientCredential) Special() string {
	if a.Name == "" {
		return SpecialCredentialState
	}
	return SpecialCredentialState + "." + a.Name
}

// label names the readout row.
func (a AmbientCredential) label() string {
	if a.Label != "" {
		return a.Label
	}
	return "credential"
}

// hint is what the readout says when it found nothing.
func (a AmbientCredential) hint() string {
	if a.Hint != "" {
		return a.Hint
	}
	// "or" for the usual precedence chain, where any one of them will do;
	// "and" where the provider insists on all of them.
	join := " or "
	if a.All {
		join = " and "
	}
	hint := "not set — export " + strings.Join(a.Env, join)
	if a.Flag != "" {
		hint += ", or paste one below"
	}
	return hint
}

// pasteHint is the placeholder on the paste box. It is the guarantee rather
// than an instruction, because the instruction ("paste a token") is already the
// label's job and the guarantee is the part a user cannot check for themselves
// without reading the command bar.
func (a AmbientCredential) pasteHint() string {
	if a.PasteHint != "" {
		return a.PasteHint
	}
	return "paste a token — it never reaches the command line"
}

// present names the variables that hold a credential right now.
//
// It returns names and never values. A name is the whole of what the launcher
// has to show, and a value it never holds is a value it cannot leak -- not into
// a rendered row, not into a cached source result, not into a crash dump.
func (a AmbientCredential) present(env EnvLookup) []string {
	var out []string
	for _, name := range a.Env {
		if v, ok := env(name); ok && strings.TrimSpace(v) != "" {
			out = append(out, name)
		}
	}
	if a.All && len(out) != len(a.Env) {
		// A partial set is not a credential: the provider needs all of them,
		// and reporting "found" off the first one would be confidently wrong.
		return nil
	}
	return out
}

// chosen turns what present found into the one line the readout shows.
func (a AmbientCredential) chosen(found []string) string {
	switch {
	case len(found) == 0:
		return ""
	case a.All:
		return strings.Join(found, " + ")
	default:
		return found[0]
	}
}

// source is the ClassAmbient source behind this credential.
//
// Reading an environment variable does not need the model's asynchronous
// machinery -- the readout is filled synchronously by RefreshAmbient, so it is
// right on the first frame rather than a tick later. The source is declared
// anyway because it is what makes this a source rather than a special case:
// it gives the credential an Activity to name while anything waits on it, a
// Cost that says when it may run, and a Prefer that agrees with the readout.
func (a AmbientCredential) source() Source {
	return Source{
		ID:    a.Source,
		Class: ClassAmbient,
		// Checking a variable is cheaper than the file reads that share this
		// cost, and it never leaves this process.
		Cost:     CostInstant,
		Activity: "checking " + strings.Join(a.Env, " and "),
		Tool:     a.Env[0],
		Prefer: func(values []string) (string, string) {
			if v := a.chosen(values); v != "" {
				return v, AmbientWhyEnv
			}
			return "", ""
		},
		Fetch: func([]string) ([]string, error) { return a.present(ambientEnv), nil },
	}
}

// readout is the credential-state field for this credential.
func (a AmbientCredential) readout() tuiform.Field {
	fd := tuiform.NewField(tuiform.Decl{
		Label:   a.label(),
		Desc:    a.hint(),
		Kind:    tuiform.KindCredentialState,
		Special: a.Special(),
		Section: tuiform.SectionCredential,
	})
	fd.SetSource(a.Source, tuiform.Emit(Emit(a.Source)))
	return fd
}

// ApplyAmbient gives a connector's ambient credentials their widgets: a readout
// for each, and the paste box that the connector's own token flag becomes.
//
// It runs after the curated overlay, so a spec's label and section for the
// token flag survive and only its kind changes, and before resolveSources,
// which is what fills the readouts in.
//
// It takes the connector's name rather than the connector, because the name is
// all it ever reads and asking for the whole record would put the connector
// catalog in this package's import list for one string.
func ApplyAmbient(f *tuiform.Form, connector string) {
	for _, a := range ambientFor(connector) {
		a.apply(f)
	}
}

func (a AmbientCredential) apply(f *tuiform.Form) {
	// Where the readout goes: immediately above the paste box it describes,
	// or at the end when the credential has no flag at all.
	at := len(f.Fields())
	if i := f.IndexOfFlag(a.Flag); i >= 0 {
		fd := &f.Fields()[i]
		fd.Kind = tuiform.KindPaste
		// The classifier already reads --token as a secret for every one of
		// these, but the launcher is not guessing here: this flag *is* the
		// credential, so it is marked as one rather than left to a heuristic.
		fd.Secret = true
		fd.Section = tuiform.SectionCredential
		fd.Desc = a.pasteHint()
		at = i
	}
	// A flag the installed provider no longer declares leaves the readout
	// standing on its own rather than taking the whole widget down with it:
	// the environment still works, and saying so is still worth a row.
	f.InsertField(at, a.readout())

	if a.Compose != nil {
		a.Compose(f, ambientEnv)
	}
}

// RefreshAmbient recomputes every credential readout on the form.
//
// It runs from resolveSources, which is called when a form is built and after
// every keystroke, so the readout is never stale: paste a token and the row
// stops naming a variable, clear it and the row names the variable again.
func RefreshAmbient(f *tuiform.Form) {
	for _, a := range ambientFor(f.Subject()) {
		i := f.IndexOfSpecial(a.Special())
		if i < 0 {
			continue
		}
		f.Fields()[i].Prefill(a.observe(f, ambientEnv))
	}
}

// observe reports what the scan will actually use, and why.
//
// A pasted token wins, because that is the precedence every one of these
// providers implements: the flag first, the variables after it. A readout that
// named a variable while a pasted token was the thing about to be used would be
// confidently wrong about the one fact it exists to state.
func (a AmbientCredential) observe(f *tuiform.Form, env EnvLookup) (state, why string) {
	if i := f.IndexOfFlag(a.Flag); i >= 0 && f.Fields()[i].IsSet() {
		return "pasted", ambientWhyPaste
	}
	if v := a.chosen(a.present(env)); v != "" {
		return v, AmbientWhyEnv
	}
	return "", ""
}

// oktaOrganization composes the okta organization the way the provider does,
// minus the bug in how the provider does it.
//
// okta's ParseCLI builds `OKTA_ORG_NAME + "." + OKTA_BASE_URL` whenever
// --organization is empty, and then guards with `if organization == ""`. With
// both variables unset the composition is the literal ".", which is not "", so
// the guard never fires: the connector proceeds with an organization of "." and
// fails later as a connection error, nowhere near its cause.
//
// The launcher therefore special-cases both-empty rather than mirroring the
// composition faithfully -- with nothing to compose from there is no
// organization, and the field is left for the user to fill. A half-composed
// value is mirrored, because that one the provider really will use, and showing
// "dev-123." in an editable field is how the user gets to notice it.
func oktaOrganization(env EnvLookup) string {
	name := strings.TrimSpace(EnvValue(env, "OKTA_ORG_NAME"))
	base := strings.TrimSpace(EnvValue(env, "OKTA_BASE_URL"))
	if name == "" && base == "" {
		return ""
	}
	return name + "." + base
}

// composeOktaOrganization prefills okta's --organization from the environment.
func composeOktaOrganization(f *tuiform.Form, env EnvLookup) {
	org := oktaOrganization(env)
	if org == "" {
		return
	}
	i := f.IndexOfFlag("organization")
	if i < 0 || f.Fields()[i].IsSet() {
		return
	}
	f.Fields()[i].SetValue(org)
	f.Fields()[i].SetPrefilled(AmbientWhyEnv)
}

// EnvValue is the value of a variable, or "" when it is unset.
func EnvValue(env EnvLookup, name string) string {
	v, _ := env(name)
	return v
}
