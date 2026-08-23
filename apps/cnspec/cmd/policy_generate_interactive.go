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
	"strings"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnspec/internal/bundle"
	"go.mondoo.com/cnspec/internal/generate"
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

// bundle.ErrUIDExists reports that a check uid is already taken in the target bundle.

// bundle.GeneratedGroupTitle is the policy group the wizard adds its checks to.

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
		provider, err := w.line("Target provider (aws, gcp, azure, os, k8s, …)", generate.GuessProvider(title))
		if err != nil {
			return err
		}
		filter, err := w.line("Asset filter", generate.DefaultFilter(provider))
		if err != nil {
			return err
		}

		w.showGrounding(title+" "+desc, provider)

		check := generate.Check{
			UID:   bundle.Slugify(title),
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
	uid, err := w.uniqueUID(file, bundle.Slugify(title))
	if err != nil {
		return err
	}

	b, err := bundle.LoadFile(file)
	if err != nil {
		return err
	}
	if err := bundle.AddCheck(b, bundle.PolicyUIDForFile(file), uid, title, desc, filter, mql); err != nil {
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
	b, err := bundle.LoadFile(file)
	if err != nil {
		return "", err
	}
	taken := bundle.QueryUIDs(b)

	for {
		uid, err := w.line("Check UID", bundle.NextFreeUID(suggested, taken))
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
