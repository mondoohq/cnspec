// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"strings"
	"time"

	deliverypkg "go.mondoo.com/cnspec/cli/launcher/delivery"
	tuiform "go.mondoo.com/cnspec/cli/tui/form"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
)

// launchPlan is everything needed to run one command: what to exec, what
// environment it needs, how to clean up after it, and anything the user should
// be told before it starts.
type launchPlan struct {
	args    []string
	env     []string
	cleanup func()
	// warn is shown to the user and printed above the command. It is not an
	// error -- the scan still runs -- but it describes a weaker guarantee than
	// the one normally offered.
	warn string
}

// launchState is the command being assembled, and what the last one left on
// disk.
//
// The two are one lifecycle: preparing covers the window between the button
// being pressed and a plan coming back, and cleanup is what that plan wrote.
// They are not part of scanState because they are about a *command* -- `shell`
// takes this path and never becomes a scan at all.
type launchState struct {
	// preparing is true while the plan for a scan is being assembled off the
	// event loop. See prepareLaunchCmd: the keychain write in there can sit on
	// an OS authentication dialog for as long as the user takes to answer it.
	preparing bool
	// cleanup removes the generated inventory once the command it fed has
	// finished. Only the tea.Exec path holds one; a background scan hands its
	// cleanup to the session instead.
	cleanup func()
}

// begin claims the launch, reporting false when one is already being prepared.
// Enter twice must not write the credential twice, nor start two scans.
func (l *launchState) begin() bool {
	if l.preparing {
		return false
	}
	l.preparing = true
	return true
}

// prepared marks the assembly over, whether or not it produced a plan.
func (l *launchState) prepared() { l.preparing = false }

// release runs the cleanup and forgets it, which is what a command that owned
// one finishing means.
func (l *launchState) release() {
	if l.cleanup != nil {
		l.cleanup()
	}
	l.cleanup = nil
}

// disown drops the cleanup without running it, for when something else has
// taken over what it would remove -- a scan session, or the process-wide
// temp-file registry on the way out. Releasing the inventory twice is harmless;
// releasing it early is not, which is why this is a second method rather than
// an argument to the first.
func (l *launchState) disown() { l.cleanup = nil }

// launchRequest is the whole of what deciding a command depends on: the form
// the user filled in, and whether they chose to keep the results here.
//
// It is a type of its own because assembling a command is not a property of the
// screen it was assembled from. Model has thirty-six fields and this reads two
// of them, which the tests were paying for: forty-three of them built a whole
// Model to ask what one form emits, and every field added to the launcher's UI
// state was a field they were implicitly constructing.
type launchRequest struct {
	form form
	// incognito is the user's choice to keep the results on this machine,
	// already resolved against whether there is a choice to make at all. See
	// verb: with no credentials there is no choice, and the flag would then be
	// a word on the command line the user did not ask for.
	incognito bool
}

// launchRequestFrom is the launcher's current answer to what would be run.
func (m Model) launchRequestFrom() launchRequest {
	return launchRequest{
		form:      m.detail.form,
		incognito: m.upstream.canToggle() && m.upstream.incognito,
	}
}

// launchArgs assembles the plan for the launcher's current form.
func (m Model) launchArgs(c Connector, a Action) (launchPlan, error) {
	return m.launchRequestFrom().plan(c, a)
}

// commandPreview is what the command bar shows, read from the same request the
// button would launch. It is on Model rather than on the pane that draws it
// because the incognito choice is not the pane's to know.
func (m Model) commandPreview(c Connector) string {
	return m.launchRequestFrom().preview(c, scanAction())
}

// verb is the command words every plan starts with: the action, the connector,
// and --incognito when the user has chosen to keep the results on this machine.
//
// The flag expresses a *choice*, not a state. With no credentials there is no
// choice to express -- scan switches to incognito on its own, and the header
// badge already says so -- and passing a redundant flag would put a word on
// every command line that the user did not ask for and cannot turn off.
func (r launchRequest) verb(c Connector, a Action) []string {
	out := []string{a.Name, c.Name}
	if r.incognito {
		out = append(out, "--incognito")
	}
	return out
}

// preview is what the command bar shows, and it is deliberately a method on
// the request rather than on the pane that draws it.
//
// The bar promises "this is what will run", and for one release it was not:
// plan() prepends --incognito through verb() when the user has asked to keep
// the results on this machine, while the preview built its own prefix out of
// []string{"cnspec", a.Name, c.Name} and had no way to know. A user who toggled
// incognito saw a command missing the flag that the launch would add. Sharing
// verb() is the whole fix, and it is why the preview cannot live somewhere that
// has the form but not the choice.
//
// One difference is real and stays: the inventory route names a file that does
// not exist yet, because writing it is what plan() does -- and the provider has
// not been asked yet either, so there is nothing more specific to say. The bar
// says so rather than showing a path that would be a lie by the time it was
// read.
//
// This runs on every keystroke, which is why it decides the route from the form
// alone. Asking the provider costs a subprocess and a gRPC round trip; doing
// that per keystroke to redraw one line would be a redraw nobody could type
// through.
func (r launchRequest) preview(c Connector, a Action) string {
	if deliveryFor(r.form) == deliverInventory {
		return "cnspec " + strings.Join(r.inventoryVerb(a, "<generated, 0600>"), " ")
	}
	return strings.Join(append([]string{"cnspec", strings.Join(r.verb(c, a), " ")},
		r.form.Args()...), " ")
}

// inventoryVerb is the command for a scan driven by a generated inventory.
//
// The connector's name is absent on purpose: the inventory carries the
// connection type, and `cnspec scan --inventory-file` reads the file instead of
// a target. Naming a connector as well would make cnspec parse a target it then
// discards -- verified live, `cnspec scan local --inventory-file <ssh asset>`
// scans the ssh asset and never touches this machine.
//
// The user's incognito choice is carried here rather than by verb() for that
// same reason, and leaving it out is what this function was extracted to stop.
// The inventory route built its own argument list and dropped the flag, so a
// user who had asked to keep the results on this machine got a scan that
// reported upstream -- the one route where the choice matters most, since it is
// the route a credential travels.
func (r launchRequest) inventoryVerb(a Action, path string) []string {
	out := []string{a.Name}
	if r.incognito {
		out = append(out, "--incognito")
	}
	return append(out, "--inventory-file", path)
}

// plan picks how the command is delivered, and returns any environment the
// child needs along with it.
//
// There are two outcomes and one of them is a refusal. A form with no
// credential becomes a command line. A form with one is handed to the
// connector's own ParseCLI, and what comes back decides the rest: the asset the
// provider built is written as an inventory, the credential it built is put in
// the OS keychain and referenced by id, and if the provider turns out not to
// have kept the secret at all the launch is refused rather than run
// unauthenticated.
func (r launchRequest) plan(c Connector, a Action) (launchPlan, error) {
	// Some of what the user chose reaches the scan through the environment
	// rather than the command line, because the connector has no flag that
	// carries it -- a chosen Kubernetes cluster is the original case. See
	// environment.
	fieldEnv, envCleanup, err := r.environment()
	if err != nil {
		return launchPlan{}, err
	}

	if deliveryFor(r.form) == deliverPlain {
		return launchPlan{
			args:    append(r.verb(c, a), r.form.Args()...),
			env:     fieldEnv,
			cleanup: envCleanup,
		}, nil
	}

	plan, err := r.inventoryPlan(c, a)
	if err != nil {
		if envCleanup != nil {
			envCleanup()
		}
		return launchPlan{}, err
	}
	// The environment a *target* needs travels with the inventory too. Nothing
	// declares both today -- no connector with an EnvSpec also has a credential
	// field -- and the branch this replaces dropped it on the floor rather than
	// pass it on, which would have been a scan of the wrong cluster the first
	// time one did.
	plan.env = fieldEnv
	if envCleanup != nil {
		inner := plan.cleanup
		plan.cleanup = func() {
			envCleanup()
			if inner != nil {
				inner()
			}
		}
	}
	return plan, nil
}

// parsedForm is the connector's own reading of a filled-in form: the asset its
// provider would connect to, and where each secret the user typed ended up
// inside it.
//
// The two travel together because the second is only meaningful about the
// first. Locate matches by value against this asset, so a placement carried
// beside a different asset would be a claim about a credential that is not
// there -- and the placements are what decide whether the keychain can hold the
// secret and what the user has to be told about the file.
type parsedForm struct {
	asset  *inventory.Asset
	placed []deliverypkg.Located
}

// parseForm asks the connector's provider what the form means, and refuses when
// the provider kept a secret nowhere at all.
//
// It touches nothing outside the process. That is what makes it the right thing
// to run on a key press that has not yet been confirmed -- the export modal
// opens on it -- and it is the same ordering inventoryPlan has always relied
// on: asking is what says whether there is a credential the keychain can hold
// at all, and saving first would put a secret in the OS store on behalf of an
// action that then gets refused.
func (r launchRequest) parseForm(c Connector) (parsedForm, error) {
	req := deliverypkg.RequestFor(r.form, c.Flags)
	asset, err := parseCLI(c.Provider, c.Name, req.Args, req.Flags)
	if err != nil {
		return parsedForm{}, errors.Wrap(err, "cnspec ui could not prepare a scan for "+c.Name)
	}

	placed := deliverypkg.Locate(r.form, asset)
	if lost := placementsWith(placed, deliverypkg.PlacedNowhere); len(lost) > 0 {
		// The provider was handed the value and did not keep it. Running anyway
		// scans unauthenticated at best; databricks is the case that makes this
		// a refusal rather than a warning, because its credential switch has no
		// default arm and the SDK then resolves DATABRICKS_TOKEN out of the
		// ambient environment -- which does not error, it scans whatever account
		// that variable names.
		return parsedForm{}, errors.New("the " + c.Name + " provider did not keep " +
			flagList(lost) + " from these settings, so the scan would run without it")
	}
	return parsedForm{asset: asset, placed: placed}, nil
}

// plaintextFlags names the flags whose value this provider keeps somewhere no
// vault reference is read from, so that whatever file is written carries them
// in the clear.
//
// It answers from what the provider did rather than from a list of connector
// names, which is the only reason the answer is right: the list everything here
// used to name -- openai, ollama, huggingface, claude -- is measurably eleven
// connectors and thirteen fields, and four of them keep a *different*
// credential on the same form in a place the keychain can hold. See
// deliverypkg.PlacedOption for the measured set and the test that prints it.
func (p parsedForm) plaintextFlags() []string {
	return placementsWith(p.placed, deliverypkg.PlacedOption)
}

// inventoryPlan saves the credential the provider built and writes the result
// as the temporary inventory a scan reads.
func (r launchRequest) inventoryPlan(c Connector, a Action) (launchPlan, error) {
	parsed, err := r.parseForm(c)
	if err != nil {
		return launchPlan{}, err
	}

	secretID, saved, keychainErr := r.saveCredential(c, parsed.placed)
	var warn string
	if keychainErr != nil {
		// Not fatal, and not silent. The scan still runs and the secret still
		// stays off the command line, but it is now written to a file rather
		// than held by the operating system, and the user should know that.
		//
		// The export path answers the same failure with a refusal, because the
		// file it would fall back to is one the user chose a path for and keeps.
		// See exportInventory.
		warn = "could not use the OS keychain (" + keychainErr.Error() +
			") — the credential is written into a temporary 0600 inventory file instead, " +
			"and removed after the scan"
	}
	if plain := parsed.plaintextFlags(); len(plain) > 0 {
		// Not a refusal. The alternatives are no better -- an environment
		// variable sits in /proc/<pid>/environ for the whole run, where this
		// file is 0600 inside a 0700 directory and is removed after it -- but
		// the user is told, by name, which value is in the clear.
		warn = joinWarnings(warn, "the "+c.Name+" provider reads "+flagList(plain)+
			" as a connection option rather than as a credential, so the OS keychain "+
			"cannot hold it: it is written in plain text into a temporary 0600 "+
			"inventory file, removed after the scan")
	}

	path, cleanup, err := writeInventory(deliverypkg.InventoryFor(c.Name, parsed.asset, saved, secretID))
	if err != nil {
		return launchPlan{}, err
	}
	return launchPlan{
		args:    r.inventoryVerb(a, path),
		cleanup: cleanup,
		warn:    warn,
	}, nil
}

// placementsWith names the form's flags whose value the provider put in one
// particular place.
func placementsWith(placed []deliverypkg.Located, want deliverypkg.Placement) []string {
	var out []string
	for _, p := range placed {
		if p.Placement == want {
			out = append(out, p.Flag)
		}
	}
	return out
}

// flagList spells a set of flags the way the user typed them, for a sentence.
func flagList(flags []string) string {
	out := make([]string, len(flags))
	for i, f := range flags {
		out[i] = "--" + f
	}
	return strings.Join(out, " and ")
}

func joinWarnings(a, b string) string {
	if a == "" {
		return b
	}
	return a + "; " + b
}

// saveCredential puts the credential the provider built in the OS keychain and
// returns the id the inventory should reference.
//
// This is not offered as a choice. Where a credential can be kept somewhere the
// operating system protects, that is simply where it goes -- asking first only
// invites the worse answer, and the toggle that used to be here appeared even
// for connectors whose credential the launcher cannot carry at all.
//
// What gets saved is the provider's own credential rather than one assembled
// here, and that is the difference this change makes: artifactory's token is a
// bearer credential, azure's certificate is a pkcs12 bundle, okta's key is a
// private_key, and hcp routes on cred.User carrying the label "client-secret".
// Every one of those used to be flattened into a password with no user, and a
// provider whose credential switch did not recognise the result dropped the
// credential without a word.
//
// An empty id and a nil error mean there was nothing for the keychain to hold:
// the form carried no credential, or the provider kept it somewhere a vault
// reference is never read from. A non-nil error means the keychain refused, and
// what to do about that is the caller's to decide -- the scan falls back to the
// temporary 0600 inventory with a warning, and an export refuses outright. That
// fallback is deliberately not an encrypted file, because the encrypted-file
// vault takes its password from Options["password"] in the same inventory that
// references it, which protects nothing while looking like it does.
func (r launchRequest) saveCredential(c Connector, placed []deliverypkg.Located) (secretID string, saved *deliverypkg.Located, err error) {
	saved = deliverypkg.Keychainable(placed)
	if saved == nil {
		return "", nil, nil
	}
	cred := saved.Credential()
	if cred == nil {
		return "", nil, nil
	}
	id := newSecretID(c.Name, cred.User, time.Now())
	if err := storeCredentialWithin(keychainTimeout, id, cred); err != nil {
		log.Debug().Err(err).Msg("keychain unavailable")
		// Nothing is returned to reference: the caller must not build an
		// inventory pointing at a keychain entry that was never written.
		return "", nil, err
	}
	if err := recordSaved(savedEntry{
		ID:        id,
		Connector: c.Name,
		Label:     cred.User,
		SavedAt:   time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		log.Debug().Err(err).Msg("could not record the saved credential")
	}
	return id, saved, nil
}

// environment is the environment the form's own values contribute to the child,
// and the cleanup for anything written on their behalf.
//
// Not every value the user picks has a flag to travel in, and that is a
// property of the connector rather than an accident. k8s --context is parsed
// and then never reaches the client config; alicloud declares no --profile;
// snowflake takes no connection name; docker and container take no --context.
// For all of them the value has to reach the child process through its
// environment instead.
//
// This used to be one function that opened `if m.form.connector != "k8s"`,
// called from one place. Four more connectors needing the same thing would have
// meant four more special cases in it, which is exactly the shape the declared
// Source and FormSpec contracts were written to end -- so the declaration moved
// onto the field, and this walks whatever the form declares.
func (r launchRequest) environment() ([]string, func(), error) {
	var env []string
	var cleanups []func()
	// The caller gets one cleanup whatever happened, including the partial
	// work of a contributor that failed after an earlier one succeeded.
	runCleanups := func() {
		for _, c := range cleanups {
			c()
		}
	}
	fail := func(err error) ([]string, func(), error) {
		runCleanups()
		return nil, nil, err
	}

	for _, fd := range r.form.Fields() {
		if !fd.IsSet() {
			continue
		}
		if v := fieldEnvVar(fd); v != "" {
			env = append(env, v+"="+fd.Emitted())
		}
		env = append(env, neutralisedBy(fieldEnvVar(fd), fd.Emitted())...)
		for _, spec := range envSpecsFor(r.form.Subject()) {
			if !tuiform.MatchesIdentity(fd, spec.Field) {
				continue
			}
			more, cleanup, err := spec.Apply(fd.Emitted())
			if cleanup != nil {
				cleanups = append(cleanups, cleanup)
			}
			if err != nil {
				return fail(err)
			}
			env = append(env, more...)
		}
	}

	if len(cleanups) == 0 {
		return env, nil, nil
	}
	return env, runCleanups, nil
}

// fieldEnvVar names the environment variable a field's value travels in: what
// the spec said, or failing that what the source attached to it declares --
// a docker context is DOCKER_CONTEXT wherever it is offered.
func fieldEnvVar(fd field) string {
	if fd.Env != "" {
		return fd.Env
	}
	if s, ok := sourceByID(fd.Source()); ok && s.Env != "" {
		return s.Env
	}
	return ""
}
