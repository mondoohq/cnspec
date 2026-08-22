// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"go.mondoo.com/cnspec/v13/internal/bundle"
	"go.mondoo.com/cnspec/v13/internal/generate"
)

// runGenerateWizard is the interactive, guided authoring experience: describe a
// check in plain language, watch cnspec generate and validate the MQL, then
// accept / edit / regenerate, and write it into a bundle — one check at a time.
func runGenerateWizard(ctx context.Context, gen *generate.Generator, defaultFile string) error {
	r := bufio.NewReader(os.Stdin)

	fmt.Fprintln(os.Stderr, "\nInteractive MQL generation")
	fmt.Fprintln(os.Stderr, "Describe a check in plain language; cnspec generates and validates the MQL.")
	fmt.Fprintln(os.Stderr, "Press Ctrl-C any time to exit.")

	if defaultFile == "" {
		defaultFile = "policy.mql.yaml"
	}
	file := promptLine(r, "Bundle file to write checks to", defaultFile)

	added := 0
	for {
		fmt.Fprintln(os.Stderr, "\n── New check ──────────────────────────────")
		title := promptRequired(r, "What should this check verify?")
		desc := promptLine(r, "More detail (optional)", "")
		provider := promptLine(r, "Target provider (aws, gcp, azure, os, k8s, …)", guessProvider(title))
		filter := promptLine(r, "Asset filter", defaultFilter(provider))

		// grounding preview: show real precedents the generator will also use
		if examples := gen.Ground(title+" "+desc, provider, 3); len(examples) > 0 {
			fmt.Fprintln(os.Stderr, "\nSimilar existing checks (used as grounding):")
			for _, e := range examples {
				name := e.Title
				if name == "" {
					name = e.UID
				}
				fmt.Fprintf(os.Stderr, "  • %s\n      %s\n", name, oneLineMQL(e.Mql, 100))
			}
		}

		check := generate.Check{
			UID:   slugify(title),
			Title: title,
			Desc:  desc,
		}
		if filter != "" {
			check.Filters = []string{filter}
		}

		mql, ok := reviewLoop(ctx, r, gen, check)
		if ok {
			uid := promptLine(r, "Check UID", slugify(title))
			if err := appendCheck(file, uid, title, desc, filter, mql); err != nil {
				fmt.Fprintf(os.Stderr, "  could not write check: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "  ✓ added %s to %s\n", uid, file)
				added++
			}
		}

		if !promptYesNo(r, "Add another check?") {
			break
		}
	}

	fmt.Fprintf(os.Stderr, "\nDone: added %d check(s) to %s\n", added, file)
	if added > 0 {
		fmt.Fprintf(os.Stderr, "Next: `cnspec policy lint %s`, then commit.\n", file)
	}
	return nil
}

// reviewLoop generates MQL for a check and drives the accept/edit/regenerate
// review. Returns the chosen MQL and whether the user accepted anything.
func reviewLoop(ctx context.Context, r *bufio.Reader, gen *generate.Generator, check generate.Check) (string, bool) {
	for {
		fmt.Fprintln(os.Stderr, "\nGenerating…")
		res := gen.GenerateCheck(ctx, check)

		if res.Action != generate.ActionGenerated {
			fmt.Fprintf(os.Stderr, "  generation failed: %s\n", res.Reason)
			switch promptChoice(r, "  [r]etry, [e]dit manually, [s]kip", "r", "e", "s") {
			case "r":
				continue
			case "e":
				if mql, ok := editAndValidate(r, gen, ""); ok {
					return mql, true
				}
				return "", false
			default:
				return "", false
			}
		}

		fmt.Fprintln(os.Stderr, "\nGenerated MQL:")
		fmt.Fprintln(os.Stderr, indentBlock(res.MQL))
		if res.Explanation != "" {
			fmt.Fprintf(os.Stderr, "  (%s)\n", res.Explanation)
		}

		switch promptChoice(r, "  [a]ccept, [e]dit, [r]egenerate with feedback, [s]kip", "a", "e", "r", "s") {
		case "a":
			return res.MQL, true
		case "e":
			if mql, ok := editAndValidate(r, gen, res.MQL); ok {
				return mql, true
			}
			// user backed out of editing; fall through to re-review
		case "r":
			check.Guidance = strings.TrimSpace(check.Guidance + " " + promptRequired(r, "  What should change?"))
		default:
			return "", false
		}
	}
}

// editAndValidate lets the user hand-edit MQL and compile-checks the result,
// offering to keep an invalid query, re-edit, or cancel.
func editAndValidate(r *bufio.Reader, gen *generate.Generator, current string) (string, bool) {
	for {
		mql := editMQL(r, current)
		if strings.TrimSpace(mql) == "" {
			return "", false
		}
		if err := gen.Validate(mql); err != nil {
			fmt.Fprintf(os.Stderr, "  does not validate: %v\n", err)
			switch promptChoice(r, "  [e]dit again, [k]eep anyway, [c]ancel", "e", "k", "c") {
			case "e":
				current = mql
				continue
			case "k":
				return mql, true
			default:
				return "", false
			}
		}
		return mql, true
	}
}

// editMQL opens $EDITOR on the current MQL, falling back to inline multi-line
// entry (terminated by a blank line) when no editor is set or it fails.
func editMQL(r *bufio.Reader, current string) string {
	if editor := strings.TrimSpace(os.Getenv("EDITOR")); editor != "" {
		if edited, err := editViaEditor(editor, current); err == nil {
			return strings.TrimSpace(edited)
		}
		fmt.Fprintln(os.Stderr, "  (editor failed)")
	}
	fmt.Fprintln(os.Stderr, "  Enter MQL, end with a blank line:")
	var lines []string
	for {
		line, err := r.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")
		if strings.TrimSpace(line) == "" {
			break
		}
		lines = append(lines, line)
		if err != nil {
			break
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
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

// appendCheck adds a new query to a bundle file (creating it if absent),
// preserving existing formatting/comments and writing atomically.
func appendCheck(file, uid, title, desc, filter, mql string) error {
	var b *bundle.Bundle
	if data, err := os.ReadFile(file); err == nil {
		if b, err = bundle.ParseYaml(data); err != nil {
			return fmt.Errorf("could not parse %s: %w", file, err)
		}
	} else if os.IsNotExist(err) {
		b = &bundle.Bundle{}
	} else {
		return err
	}

	q := &bundle.Mquery{Uid: uid, Title: title, Mql: mql}
	if strings.TrimSpace(filter) != "" {
		q.Filters = &bundle.Filters{Items: map[string]*bundle.Mquery{"": {Mql: filter}}}
	}
	if strings.TrimSpace(desc) != "" {
		q.Docs = &bundle.MqueryDocs{Desc: desc}
	}
	b.Queries = append(b.Queries, q)

	out, err := bundle.FormatBundle(b, false)
	if err != nil {
		return err
	}
	return writeFileAtomic(file, out)
}

// --- small prompt + text helpers -------------------------------------------

func promptLine(r *bufio.Reader, label, def string) string {
	if def != "" {
		fmt.Fprintf(os.Stderr, "%s [%s]: ", label, def)
	} else {
		fmt.Fprintf(os.Stderr, "%s: ", label)
	}
	s, _ := r.ReadString('\n')
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	return s
}

func promptRequired(r *bufio.Reader, label string) string {
	for {
		if s := promptLine(r, label, ""); s != "" {
			return s
		}
		fmt.Fprintln(os.Stderr, "  (required)")
	}
}

func promptYesNo(r *bufio.Reader, msg string) bool {
	s := strings.ToLower(promptLine(r, msg+" [Y/n]", "y"))
	return s == "y" || s == "yes"
}

func promptChoice(r *bufio.Reader, label string, opts ...string) string {
	for {
		s := strings.ToLower(promptLine(r, label, opts[0]))
		for _, o := range opts {
			if s == o {
				return o
			}
		}
		fmt.Fprintf(os.Stderr, "  choose one of: %s\n", strings.Join(opts, ", "))
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

// defaultFilter proposes an asset filter for a provider.
func defaultFilter(provider string) string {
	switch provider {
	case "":
		return ""
	case "os":
		return `asset.family.contains("linux")`
	default:
		return fmt.Sprintf(`asset.platform == %q`, provider)
	}
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

func oneLineMQL(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}
