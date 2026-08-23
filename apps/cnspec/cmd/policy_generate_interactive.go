// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnspec/internal/bundle"
	"go.mondoo.com/cnspec/internal/generate"
	"go.mondoo.com/mql/providers"
)

// errInputClosed reports that the wizard's input stream ended (EOF: Ctrl-D, a
// closed pipe, or a script that ran out of answers).
//
// Every prompt must surface this instead of returning its default. When EOF fell
// through to the default, `promptRequired` re-printed its prompt forever (tens of
// megabytes of stderr per second), `promptChoice` picked opts[0] — `[r]etry` on a
// failed generation, which re-invoked the coding agent in a tight unattended
// loop, and `[a]ccept` on a successful one, which wrote MQL no human ever saw.
var errInputClosed = errors.New("input closed")

// errUIDExists reports that a check uid is already taken in the target bundle.
var errUIDExists = errors.New("uid already used in this bundle")

// generatedGroupTitle is the policy group the wizard adds its checks to.
const generatedGroupTitle = "Generated checks"

// wizardOpts is everything the wizard needs from the command line. In and Out
// are injected so the flow can be driven by a test with scripted stdin.
type wizardOpts struct {
	In  io.Reader
	Out io.Writer
	// File is the bundle to write checks into. FileFromFlag marks it as an
	// explicit --output, which is used as-is instead of being offered as the
	// default of a prompt.
	File         string
	FileFromFlag bool
	// DryRun previews each accepted check instead of writing it.
	DryRun bool
}

type wizard struct {
	in    *bufio.Reader
	out   io.Writer
	gen   *generate.Generator
	opts  wizardOpts
	added int
}

// runGenerateWizard is the interactive, guided authoring experience: describe a
// check in plain language, watch cnspec generate and validate the MQL, then
// accept / edit / regenerate, and write it into a bundle — one check at a time.
func runGenerateWizard(ctx context.Context, gen *generate.Generator, opts wizardOpts) error {
	if opts.In == nil {
		opts.In = os.Stdin
	}
	if opts.Out == nil {
		opts.Out = os.Stderr
	}
	w := &wizard{in: bufio.NewReader(opts.In), out: opts.Out, gen: gen, opts: opts}
	return w.run(ctx)
}

func (w *wizard) run(ctx context.Context) error {
	fmt.Fprintln(w.out, "\nInteractive MQL generation")
	fmt.Fprintln(w.out, "Describe a check in plain language; cnspec generates and validates the MQL.")
	fmt.Fprintln(w.out, "Press Ctrl-C any time to exit.")
	if w.opts.DryRun {
		fmt.Fprintln(w.out, "--dry-run: accepted checks are previewed, not written.")
	}

	file, err := w.targetFile()
	if err == nil {
		err = w.loop(ctx, file)
	}

	verb := "added"
	if w.opts.DryRun {
		verb = "would add"
	}
	if file != "" {
		fmt.Fprintf(w.out, "\nDone: %s %d check(s) to %s\n", verb, w.added, file)
		if w.added > 0 && !w.opts.DryRun {
			fmt.Fprintf(w.out, "Next: `cnspec policy lint %s` and `cnspec scan <target> --policy-bundle %s`, then commit.\n", file, file)
		}
	}

	// EOF is a legitimate way to end a session (Ctrl-D, or a script whose input
	// ran out); it is not an error, but it must never be mistaken for an answer.
	if errors.Is(err, errInputClosed) {
		fmt.Fprintln(w.out, "Input ended; nothing was written for the check in progress.")
		return nil
	}
	return err
}

// targetFile resolves the bundle the wizard writes to: an explicit --output is
// taken as given, anything else is a prompt seeded with the path argument.
func (w *wizard) targetFile() (string, error) {
	if w.opts.FileFromFlag {
		return w.opts.File, nil
	}
	def := w.opts.File
	if def == "" {
		def = "policy.mql.yaml"
	}
	return w.line("Bundle file to write checks to", def)
}

func (w *wizard) loop(ctx context.Context, file string) error {
	for {
		fmt.Fprintln(w.out, "\n── New check ──────────────────────────────")
		title, err := w.required("What should this check verify?")
		if err != nil {
			return err
		}
		desc, err := w.line("More detail (optional)", "")
		if err != nil {
			return err
		}
		provider, err := w.line("Target provider (aws, gcp, azure, os, k8s, …)", guessProvider(title))
		if err != nil {
			return err
		}
		filter, err := w.line("Asset filter", defaultFilter(provider))
		if err != nil {
			return err
		}

		w.showGrounding(title+" "+desc, provider)

		check := generate.Check{
			UID:   slugify(title),
			Title: title,
			Desc:  desc,
		}
		if filter != "" {
			check.Filters = []string{filter}
		}

		mql, ok, err := w.review(ctx, check)
		if err != nil {
			return err
		}
		if ok {
			if err := w.writeCheck(file, title, desc, filter, mql); err != nil {
				if errors.Is(err, errInputClosed) {
					return err
				}
				fmt.Fprintf(w.out, "  could not write check: %v\n", err)
			}
		}

		again, err := w.yesNo("Add another check?")
		if err != nil {
			return err
		}
		if !again {
			return nil
		}
	}
}

// showGrounding previews the real precedents the generator also feeds the agent.
func (w *wizard) showGrounding(intent, provider string) {
	examples := w.gen.Ground(intent, provider, 3)
	if len(examples) == 0 {
		return
	}
	fmt.Fprintln(w.out, "\nSimilar existing checks (used as grounding):")
	for _, e := range examples {
		name := e.Title
		if name == "" {
			name = e.UID
		}
		fmt.Fprintf(w.out, "  • %s\n      %s\n", name, oneLineMQL(e.Mql, 100))
	}
}

// writeCheck asks for the uid and adds the accepted check to the bundle, or
// previews it under --dry-run.
func (w *wizard) writeCheck(file, title, desc, filter, mql string) error {
	uid, err := w.uniqueUID(file, slugify(title))
	if err != nil {
		return err
	}

	b, err := loadBundleFile(file)
	if err != nil {
		return err
	}
	if err := addCheck(b, policyUIDForFile(file), uid, title, desc, filter, mql); err != nil {
		return err
	}
	out, err := bundle.FormatBundle(b, false)
	if err != nil {
		return err
	}

	if w.opts.DryRun {
		fmt.Fprintf(w.out, "\n--- %s (dry run, not written) ---\n%s\n", file, out)
		w.added++
		return nil
	}
	if err := writeFileAtomic(file, out); err != nil {
		return err
	}
	fmt.Fprintf(w.out, "  ✓ added %s to %s\n", uid, file)
	w.added++
	return nil
}

// uniqueUID prompts for the check uid and refuses one that is already defined in
// the target bundle: appending a duplicate produces a bundle that fails
// `cnspec policy lint` with query-uid-unique, which the user only finds later.
func (w *wizard) uniqueUID(file, suggested string) (string, error) {
	b, err := loadBundleFile(file)
	if err != nil {
		return "", err
	}
	taken := bundle.QueryUIDs(b)

	for {
		uid, err := w.line("Check UID", nextFreeUID(suggested, taken))
		if err != nil {
			return "", err
		}
		if uid == "" {
			fmt.Fprintln(w.out, "  (a check needs a uid)")
			continue
		}
		if !taken[uid] {
			return uid, nil
		}
		fmt.Fprintf(w.out, "  uid %q is already used in %s; pick another.\n", uid, file)
		suggested = uid
	}
}

// nextFreeUID suffixes a uid until it is free, so the prompt's default is one
// the user can accept rather than a collision they have to resolve by hand.
func nextFreeUID(uid string, taken map[string]bool) string {
	if uid == "" || !taken[uid] {
		return uid
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", uid, i)
		if !taken[candidate] {
			return candidate
		}
	}
}

// review generates MQL for a check and drives the accept/edit/regenerate review.
// Returns the chosen MQL and whether the user accepted anything.
func (w *wizard) review(ctx context.Context, check generate.Check) (string, bool, error) {
	for {
		fmt.Fprintln(w.out, "\nGenerating…")
		res := w.gen.GenerateCheck(ctx, check)

		if res.Action != generate.ActionGenerated {
			fmt.Fprintf(w.out, "  generation failed: %s\n", bundle.SanitizeText(res.Reason))
			choice, err := w.choice("  [r]etry, [e]dit manually, [s]kip", "r", "e", "s")
			if err != nil {
				return "", false, err
			}
			switch choice {
			case "r":
				continue
			case "e":
				return w.editAndValidate("", check.Props)
			default:
				return "", false, nil
			}
		}

		// render the agent's output through the same normalization that writes it
		// to disk, so the text under review is the text that gets committed
		mql := bundle.SanitizeText(res.MQL)
		explanation := bundle.SanitizeText(res.Explanation)

		regenerate := false
		for !regenerate {
			fmt.Fprintln(w.out, "\nGenerated MQL:")
			fmt.Fprintln(w.out, indentBlock(mql))
			if explanation != "" {
				fmt.Fprintf(w.out, "  (%s)\n", explanation)
			}

			choice, err := w.choice("  [a]ccept, [e]dit, [r]egenerate with feedback, [s]kip", "a", "e", "r", "s")
			if err != nil {
				return "", false, err
			}
			switch choice {
			case "a":
				return mql, true, nil
			case "e":
				// Backing out of the editor returns to *this* candidate. Falling
				// through to the outer loop would call the agent again — a second
				// billed invocation that replaces the query the user was reviewing
				// with a different one.
				edited, ok, err := w.editAndValidate(mql, check.Props)
				if err != nil {
					return "", false, err
				}
				if ok {
					return edited, true, nil
				}
			case "r":
				feedback, err := w.required("  What should change?")
				if err != nil {
					return "", false, err
				}
				check.Guidance = strings.TrimSpace(check.Guidance + " " + feedback)
				regenerate = true
			default:
				return "", false, nil
			}
		}
	}
}

// editAndValidate lets the user hand-edit MQL and compile-checks the result,
// offering to keep an invalid query, re-edit, or cancel.
func (w *wizard) editAndValidate(current string, props []generate.Prop) (string, bool, error) {
	for {
		mql, err := w.editMQL(current)
		if err != nil {
			return "", false, err
		}
		if strings.TrimSpace(mql) == "" {
			return "", false, nil
		}
		if err := w.gen.Validate(mql, props...); err != nil {
			fmt.Fprintf(w.out, "  does not validate: %v\n", err)
			choice, cerr := w.choice("  [e]dit again, [k]eep anyway, [c]ancel", "e", "k", "c")
			if cerr != nil {
				return "", false, cerr
			}
			switch choice {
			case "e":
				current = mql
				continue
			case "k":
				return mql, true, nil
			default:
				return "", false, nil
			}
		}
		return mql, true, nil
	}
}

// editMQL opens $EDITOR on the current MQL, falling back to inline multi-line
// entry (terminated by a blank line) when no editor is set or it fails.
func (w *wizard) editMQL(current string) (string, error) {
	if editor := strings.TrimSpace(os.Getenv("EDITOR")); editor != "" {
		if edited, err := editViaEditor(editor, current); err == nil {
			return strings.TrimSpace(edited), nil
		}
		fmt.Fprintln(w.out, "  (editor failed)")
	}
	fmt.Fprintln(w.out, "  Enter MQL, end with a blank line:")
	var lines []string
	for {
		line, err := w.rawLine()
		if err != nil {
			// EOF ends the entry with whatever was typed; it must not be read as
			// an endless supply of blank lines.
			if errors.Is(err, errInputClosed) && len(lines) > 0 {
				break
			}
			return "", err
		}
		if strings.TrimSpace(line) == "" {
			break
		}
		lines = append(lines, line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n")), nil
}

func editViaEditor(editor, current string) (string, error) {
	f, err := os.CreateTemp("", "cnspec-mql-*.mql")
	if err != nil {
		return "", err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err := f.WriteString(current); err != nil {
		f.Close()
		return "", err
	}
	f.Close()

	// EDITOR may include arguments (e.g. "code --wait")
	parts := strings.Fields(editor)
	cmd := exec.Command(parts[0], append(parts[1:], name)...) //nolint:gosec // user's own $EDITOR
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	out, err := os.ReadFile(name)
	return string(out), err
}

// --- bundle writing ---------------------------------------------------------

// loadBundleFile parses a bundle file, returning an empty bundle when the file
// does not exist yet.
func loadBundleFile(file string) (*bundle.Bundle, error) {
	data, err := os.ReadFile(file)
	if os.IsNotExist(err) {
		return &bundle.Bundle{}, nil
	}
	if err != nil {
		return nil, err
	}
	b, err := bundle.ParseYaml(data)
	if err != nil {
		return nil, errors.Wrapf(err, "could not parse %s", file)
	}
	return b, nil
}

// appendCheck adds a new check to a bundle file (creating it if absent),
// preserving existing formatting/comments and writing atomically.
func appendCheck(file, uid, title, desc, filter, mql string) error {
	b, err := loadBundleFile(file)
	if err != nil {
		return err
	}
	if err := addCheck(b, policyUIDForFile(file), uid, title, desc, filter, mql); err != nil {
		return err
	}
	out, err := bundle.FormatBundle(b, false)
	if err != nil {
		return err
	}
	return writeFileAtomic(file, out)
}

// addCheck places a check inside a policy group.
//
// A check that lives only in the top-level `queries:` block is not scannable:
// lint reports it as query-unassigned and `cnspec scan --policy-bundle` then
// fails with "a policy or framework mrn is required". The wizard tells the user
// to lint and scan what it wrote, so what it writes has to survive both.
func addCheck(b *bundle.Bundle, policyUID, uid, title, desc, filter, mql string) error {
	if b == nil {
		return errors.New("no bundle to add the check to")
	}
	if strings.TrimSpace(uid) == "" {
		return errors.New("a check needs a uid")
	}
	if bundle.QueryUIDs(b)[uid] {
		return errors.Wrapf(errUIDExists, "check %q", uid)
	}

	q := &bundle.Mquery{Uid: uid, Title: title, Mql: mql}
	if strings.TrimSpace(filter) != "" {
		q.Filters = &bundle.Filters{Items: map[string]*bundle.Mquery{"": {Mql: filter}}}
	}
	if strings.TrimSpace(desc) != "" {
		q.Docs = &bundle.MqueryDocs{Desc: desc}
	}

	group := generatedGroup(targetPolicy(b, policyUID))
	group.Checks = append(group.Checks, q)
	return nil
}

// targetPolicy returns the policy new checks join: the bundle's first policy
// when it has one, otherwise a new minimal policy (uid + name + semver version
// are what `cnspec policy lint` requires).
func targetPolicy(b *bundle.Bundle, policyUID string) *bundle.Policy {
	if len(b.Policies) > 0 && b.Policies[0] != nil {
		return b.Policies[0]
	}
	p := &bundle.Policy{
		Uid:     policyUID,
		Name:    humanize(policyUID),
		Version: "1.0.0",
	}
	b.Policies = append(b.Policies, p)
	return p
}

// generatedGroup returns the group new checks are appended to, creating it when
// needed.
//
// It is deliberately a group of its own rather than the policy's first group: a
// group-level filter gates every check inside it, so dropping a fresh check into
// an existing group can silently restrict it to assets it was never written for
// (a group filtered to hosts running kube-apiserver, say). The checks carry
// their own filters, which is what policy-missing-asset-filter asks for.
func generatedGroup(p *bundle.Policy) *bundle.PolicyGroup {
	for _, g := range p.Groups {
		if g != nil && g.Title == generatedGroupTitle && (g.Filters == nil || len(g.Filters.Items) == 0) {
			return g
		}
	}
	g := &bundle.PolicyGroup{Title: generatedGroupTitle}
	p.Groups = append(p.Groups, g)
	return g
}

// policyUidRe mirrors the uid requirement `cnspec policy lint` enforces on a
// policy (bundle-invalid-uid): 4-100 characters of lowercase letters, digits,
// dashes and underscores.
var policyUidRe = regexp.MustCompile(`^([\d\-_]|[a-z]){4,100}$`)

// policyUIDForFile derives the uid of the policy the wizard creates from the
// bundle's file name, falling back to a generic uid when that would not lint.
func policyUIDForFile(file string) string {
	base := filepath.Base(file)
	base = strings.TrimSuffix(base, ".yaml")
	base = strings.TrimSuffix(base, ".yml")
	base = strings.TrimSuffix(base, ".mql")
	uid := slugify(base)
	if !policyUidRe.MatchString(uid) {
		return "generated-policy"
	}
	return uid
}

// humanize turns a uid into a policy name ("aws-s3" -> "Aws S3").
func humanize(uid string) string {
	fields := strings.FieldsFunc(uid, func(r rune) bool { return r == '-' || r == '_' })
	for i, f := range fields {
		fields[i] = strings.ToUpper(f[:1]) + f[1:]
	}
	if len(fields) == 0 {
		return uid
	}
	return strings.Join(fields, " ")
}

// --- small prompt + text helpers -------------------------------------------

// rawLine reads one line, reporting errInputClosed at EOF. A final line without
// a trailing newline is still an answer; the EOF surfaces on the next read.
func (w *wizard) rawLine() (string, error) {
	s, err := w.in.ReadString('\n')
	s = strings.TrimRight(s, "\r\n")
	if err != nil {
		if strings.TrimSpace(s) == "" {
			return "", errInputClosed
		}
		return s, nil
	}
	return s, nil
}

func (w *wizard) line(label, def string) (string, error) {
	if def != "" {
		fmt.Fprintf(w.out, "%s [%s]: ", label, def)
	} else {
		fmt.Fprintf(w.out, "%s: ", label)
	}
	s, err := w.rawLine()
	if err != nil {
		return "", err
	}
	if s = strings.TrimSpace(s); s == "" {
		return def, nil
	}
	return s, nil
}

func (w *wizard) required(label string) (string, error) {
	for {
		s, err := w.line(label, "")
		if err != nil {
			return "", err
		}
		if s != "" {
			return s, nil
		}
		fmt.Fprintln(w.out, "  (required)")
	}
}

func (w *wizard) yesNo(msg string) (bool, error) {
	s, err := w.line(msg+" [Y/n]", "y")
	if err != nil {
		return false, err
	}
	s = strings.ToLower(s)
	return s == "y" || s == "yes", nil
}

func (w *wizard) choice(label string, opts ...string) (string, error) {
	for {
		s, err := w.line(label, opts[0])
		if err != nil {
			return "", err
		}
		s = strings.ToLower(s)
		for _, o := range opts {
			if s == o {
				return o, nil
			}
		}
		fmt.Fprintf(w.out, "  choose one of: %s\n", strings.Join(opts, ", "))
	}
}

// guessProvider makes a light guess at the target provider from the title so the
// prompt can offer a sensible default.
func guessProvider(title string) string {
	t := strings.ToLower(title)
	switch {
	case containsAny(t, "aws", "s3", "ec2", "iam", "cloudtrail", "rds", "eks", "lambda"):
		return "aws"
	case containsAny(t, "gcp", "gke", "bigquery", "cloud sql", "compute engine"):
		return "gcp"
	case containsAny(t, "azure", "aks", "blob"):
		return "azure"
	case containsAny(t, "ssh", "sshd", "kernel", "linux", "package", "systemd", "sudo", "pam"):
		return "os"
	case containsAny(t, "kubernetes", "k8s", "pod", "container"):
		return "k8s"
	}
	return ""
}

// curatedFilters maps a provider to the asset filter the wizard proposes.
//
// These are *platform* names, not provider names, and the two differ for most
// providers. A filter naming a platform that does not exist is dead: it lints
// clean, it scans clean, and it never matches an asset — which is what
// `asset.platform == "gcp"` and `asset.platform == "k8s"` were, since neither
// name exists (gcp's platforms are gcp-project, gcp-gke-cluster, …; "k8s" is a
// platform *family*, and its platforms are k8s-cluster, k8s-pod, …).
//
// Every name here is taken from installed provider metadata
// (~/.config/mondoo/providers/<n>/<n>.json, Platforms[].name and .family) and is
// in use by real checks in content/ — see TestDefaultFilterUsesRealPlatformNames,
// which re-derives both. The choice among a provider's platforms is the scope
// where that provider's account-wide resources resolve.
var curatedFilters = map[string]string{
	// the account-level asset; content/ uses it for ~200 checks
	"aws": `asset.platform == "aws"`,
	// the subscription-level asset
	"azure": `asset.platform == "azure"`,
	// there is no platform named "gcp"; the project is the account-level asset
	"gcp": `asset.platform == "gcp-project"`,
	// cluster and manifest are the two scopes where the cluster-wide k8s.*
	// resources resolve; workload platforms (k8s-pod, k8s-deployment, …) are
	// per-object and too narrow to guess at
	"k8s": `asset.platform == "k8s-cluster" || asset.platform == "k8s-manifest"`,
	// os platforms are per-distribution (ubuntu, redhat, …); the family is the
	// portable way to say "any Linux"
	"os": `asset.family.contains("linux")`,
}

// defaultFilter proposes an asset filter for a provider: a curated one where the
// provider has many platforms and only some are plausible, otherwise one derived
// from the installed provider metadata. Returns "" when neither knows the
// provider — an empty prompt the user fills in beats a filter that silently
// matches nothing.
func defaultFilter(provider string) string {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return ""
	}
	if f, ok := curatedFilters[provider]; ok {
		return f
	}
	name, family := lookupPlatform(provider)
	switch {
	case name != "":
		return fmt.Sprintf(`asset.platform == %q`, name)
	case family != "":
		return fmt.Sprintf(`asset.family.contains(%q)`, family)
	}
	return ""
}

// installedPlatforms reports the platform names and families a provider
// declares. It is a variable so tests can drive defaultFilter without depending
// on which providers happen to be installed.
var installedPlatforms = func(provider string) (names []string, families []string) {
	// ListAll only reads the installed provider metadata: no network, no
	// installs, and resource schemas stay unparsed.
	all, err := providers.ListAll()
	if err != nil {
		return nil, nil
	}
	seen := map[string]bool{}
	for _, p := range all {
		if p == nil || p.Provider == nil || p.Name != provider {
			continue
		}
		for _, pl := range p.Platforms {
			if pl == nil {
				continue
			}
			names = append(names, pl.Name)
			for _, f := range pl.Family {
				if !seen[f] {
					seen[f] = true
					families = append(families, f)
				}
			}
		}
	}
	return names, families
}

// lookupPlatform resolves a provider name against installed metadata: an exact
// platform of that name (aws, digitalocean, oci, …), else a family of that name
// (github, terraform, …). Anything more ambiguous is left to the user.
func lookupPlatform(provider string) (name, family string) {
	names, families := installedPlatforms(provider)
	for _, n := range names {
		if n == provider {
			return n, ""
		}
	}
	for _, f := range families {
		if f == provider {
			return "", f
		}
	}
	return "", ""
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// slugify turns a title into a lowercase, dash-separated uid.
func slugify(s string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func indentBlock(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i := range lines {
		lines[i] = "    " + lines[i]
	}
	return strings.Join(lines, "\n")
}

// oneLineMQL collapses a query to a single display line. It sanitizes because
// its input is bundle text from the grounding corpus, which cnspec does not
// author: a control character in there would otherwise reach the terminal.
func oneLineMQL(s string, max int) string {
	s = strings.Join(strings.Fields(bundle.SanitizeText(s)), " ")
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}
