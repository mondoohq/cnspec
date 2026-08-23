// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnspec/internal/bundle"
	"go.mondoo.com/cnspec/internal/generate"
	"go.mondoo.com/cnspec/policy"
)

func init() {
	policyGenerateCmd.Flags().StringP("output", "o", "", "Write the result to this file (default: stdout, unless --in-place)")
	policyGenerateCmd.Flags().Bool("in-place", false, "Modify the input file(s) directly")
	policyGenerateCmd.Flags().Bool("dry-run", false, "Preview what would be generated without writing anything")
	policyGenerateCmd.Flags().Bool("force", false, "Regenerate queries that already have mql (overwrites existing MQL)")
	policyGenerateCmd.Flags().Bool("explain", false, "Include the agent's reasoning for each generated query")
	policyGenerateCmd.Flags().String("agent", "", "Coding agent backend: "+strings.Join(generate.BackendNames(), ", ")+" (default: auto-detect)")
	policyGenerateCmd.Flags().String("model", "", "Model to pass through to the agent CLI")
	policyGenerateCmd.Flags().String("corpus", "", "Path to policy bundles used as grounding examples (default: ./content if present, else the input files)")
	policyGenerateCmd.Flags().Bool("no-validate", false, "Skip in-process MQL compile validation of generated queries")
	policyGenerateCmd.Flags().String("test-target", "", "Execute-and-assert generated MQL against this live target (e.g. local, aws, gcp, azure): requires each query to resolve to a concrete true/false verdict, not just compile")
	policyGenerateCmd.Flags().String("test-recording", "", "Execute-and-assert generated MQL against a recording file (reproducible, no live credentials)")
	policyGenerateCmd.Flags().BoolP("interactive", "i", false, "Guided authoring: describe a check, generate, review (accept/edit/regenerate), and write it out")
	policyCmd.AddCommand(policyGenerateCmd)
}

var policyGenerateCmd = &cobra.Command{
	Use:   "generate [path ...]",
	Short: "Generate MQL for policy checks from their title and description",
	Long: `Generate MQL for policy checks that have a title and description but no mql yet.

cnspec drives the workflow — reading each check's intent, resolving its target
provider from filters, searching similar existing checks for grounding, and
validating the result — and delegates the model call to a coding-agent CLI you
already have installed and authenticated (Claude Code, Codex, Kimi, DeepSeek).
cnspec ships no LLM SDK and stores no API keys; the agent brings its own model.

Checks that already have mql are left untouched unless you pass --force. The
generated MQL is validated by compiling it in-process before it is written.

Security note: each check's title and description are sent to the coding-agent
CLI as its prompt. Because that agent can run tools and commands, only run this
in a directory you trust, on bundles you trust — a crafted description is
prompt-injection input to your agent, and the grounding corpus and agent skill
files are resolved relative to the working directory.

Examples:
  cnspec policy generate policy.mql.yaml --in-place
  cnspec policy generate policy.mql.yaml --agent codex --explain
  cnspec policy generate ./content --dry-run
  cnspec policy generate --interactive          # guided, one check at a time`,
	Args: cobra.ArbitraryArgs,
	RunE: runPolicyGenerate,
}

func runPolicyGenerate(cmd *cobra.Command, args []string) error {
	inPlace, _ := cmd.Flags().GetBool("in-place")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	explain, _ := cmd.Flags().GetBool("explain")
	noValidate, _ := cmd.Flags().GetBool("no-validate")
	agentName, _ := cmd.Flags().GetString("agent")
	model, _ := cmd.Flags().GetString("model")
	outputFile, _ := cmd.Flags().GetString("output")
	corpusPath, _ := cmd.Flags().GetString("corpus")
	testTarget, _ := cmd.Flags().GetString("test-target")
	testRecording, _ := cmd.Flags().GetString("test-recording")
	interactive, _ := cmd.Flags().GetBool("interactive")

	if testTarget != "" && testRecording != "" {
		return fmt.Errorf("use only one of --test-target or --test-recording")
	}
	if noValidate && (testTarget != "" || testRecording != "") {
		return fmt.Errorf("--no-validate cannot be combined with --test-target/--test-recording (they request validation)")
	}

	// Guided authoring when explicitly requested, or when no path is given and we
	// have an interactive terminal.
	interactive = interactive || (len(args) == 0 && isatty.IsTerminal(os.Stdin.Fd()))

	// The guided flow writes each accepted check into the bundle file you name,
	// which is already "in place"; --in-place has nothing to modify there.
	// Accepting it silently (as it did) reads as if it took effect. Note this is
	// reachable without typing -i: it turns itself on for a bare tty run.
	if interactive && inPlace {
		return fmt.Errorf("--in-place cannot be combined with the guided flow (--interactive); the wizard writes into the bundle file you name")
	}

	backend, err := generate.Backend(agentName)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Using agent backend: %s\n", backend.Name())

	var validator generate.Validator
	switch {
	case noValidate:
		validator = generate.NoopValidator{}
		fmt.Fprintln(os.Stderr, "Warning: MQL validation disabled (--no-validate); generated queries are not compile-checked.")
	case testTarget != "" || testRecording != "":
		var runtime generate.RuntimeLike
		var cleanup func()
		var label string
		if testRecording != "" {
			runtime, cleanup, err = generate.ConnectRecording(testRecording)
			label = "recording " + testRecording
		} else {
			// Executing agent-generated MQL against a live target runs it before
			// human review; on the os provider that includes command()/file().
			// Only safe with trusted bundles. --test-recording avoids this.
			fmt.Fprintf(os.Stderr, "Warning: --test-target %q executes generated MQL against a live system before you review it; only use it with policy bundles you trust.\n", testTarget)
			runtime, cleanup, err = generate.ConnectTarget(testTarget)
			label = "target " + testTarget
		}
		if err != nil {
			return fmt.Errorf("could not connect to %s: %w", label, err)
		}
		defer cleanup()
		validator, err = generate.NewExecuteValidator(generate.NewRuntimeRunner(runtime), nil)
		if err != nil {
			return fmt.Errorf("could not initialize execute-and-assert validation: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Validation: compile + execute-and-assert against %s\n", label)
	default:
		validator, err = generate.NewCompileValidator()
		if err != nil {
			validator = generate.NoopValidator{}
			log.Warn().Err(err).Msg("could not initialize MQL validation; generated queries will not be compile-checked")
		}
	}

	corpus := loadGenerateCorpus(corpusPath, args)

	gen, err := generate.New(generate.Config{
		Backend:    backend,
		Corpus:     corpus,
		Validator:  validator,
		Model:      model,
		Force:      force,
		Explain:    explain,
		SkillPaths: findSkills(),
	})
	if err != nil {
		return err
	}

	if interactive {
		opts := wizardOpts{
			In:     cmd.InOrStdin(),
			Out:    os.Stderr,
			DryRun: dryRun,
		}
		switch {
		case outputFile != "":
			// --output names the bundle to write to, so don't ask for it again
			opts.File, opts.FileFromFlag = outputFile, true
		case len(args) > 0:
			opts.File = args[0]
		}
		return runGenerateWizard(cmd.Context(), gen, opts)
	}

	files, err := policy.WalkPolicyBundleFiles(args...)
	if err != nil {
		return fmt.Errorf("could not find bundle files: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("no .mql.yaml bundle files found; pass a bundle path or use --interactive")
	}
	if len(files) > 1 && outputFile != "" {
		return fmt.Errorf("--output cannot be used with multiple input files; use --in-place")
	}
	if len(files) > 1 && !inPlace && !dryRun && outputFile == "" {
		return fmt.Errorf("multiple input files require --in-place (or --dry-run to preview)")
	}

	var totalGenerated, totalFailed int
	var fileErrs []string
	for _, f := range files {
		// A hard error on one file must not abort the batch or leave the user
		// guessing which files were written; record it and continue.
		g, failed, ferr := generateForFile(cmd.Context(), gen, f, force, dryRun, inPlace, outputFile, explain)
		totalGenerated += g
		totalFailed += failed
		if ferr != nil {
			fmt.Fprintf(os.Stderr, "  error on %s: %v\n", f, ferr)
			fileErrs = append(fileErrs, f+": "+ferr.Error())
		}
	}

	fmt.Fprintf(os.Stderr, "\nDone: %d generated, %d failed across %d file(s)\n", totalGenerated, totalFailed, len(files))
	if len(fileErrs) > 0 {
		return fmt.Errorf("%d file(s) errored:\n  %s", len(fileErrs), strings.Join(fileErrs, "\n  "))
	}
	if totalFailed > 0 {
		return fmt.Errorf("%d check(s) failed to generate", totalFailed)
	}
	return nil
}

func generateForFile(ctx context.Context, gen *generate.Generator, file string, force, dryRun, inPlace bool, outputFile string, explain bool) (generated, failed int, err error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return 0, 0, fmt.Errorf("could not read %s: %w", file, err)
	}
	raw, err := bundle.ParseYaml(data)
	if err != nil {
		return 0, 0, fmt.Errorf("could not parse %s: %w", file, err)
	}

	// Enumerate query definitions, skipping uid-only references (a policy group's
	// `checks:` entry that points at a top-level definition would otherwise be
	// processed twice). We keep the query pointers aligned with the checks so
	// generated MQL is written back to the exact definition. Variant leaves
	// inherit their intent from the parent that declares them.
	variantParents := bundle.VariantParents(raw)
	var targets []*bundle.Mquery
	var checks []generate.Check
	for _, q := range bundle.AllQueries(raw) {
		title := q.Title
		desc := bundle.QueryDesc(q)
		props := bundleProps(q)
		if strings.TrimSpace(title) == "" && strings.TrimSpace(desc) == "" {
			if parent, ok := variantParents[q.Uid]; ok {
				title = parent.Title
				desc = bundle.QueryDesc(parent)
				// the leaf's intent came from the parent; the props its wording
				// references live on the parent too, so inherit them when the leaf
				// declares none of its own.
				if len(props) == 0 {
					props = bundleProps(parent)
				}
			}
		}
		if strings.TrimSpace(title) == "" && strings.TrimSpace(desc) == "" && strings.TrimSpace(q.Mql) == "" {
			continue // pure reference or empty stub — nothing to generate or write
		}
		targets = append(targets, q)
		checks = append(checks, generate.Check{
			UID:         q.Uid,
			Title:       title,
			Desc:        desc,
			Filters:     bundle.QueryFilterStrings(q),
			Mql:         q.Mql,
			Props:       props,
			HasVariants: bundle.QueryHasVariants(q),
		})
	}

	needGen := 0
	for _, c := range checks {
		willGenerate := !c.HasVariants && (strings.TrimSpace(c.Mql) == "" || force)
		if willGenerate {
			needGen++
		}
	}

	// stdout is the output *file* in this mode (`generate in.yaml > out.yaml`), so
	// it has to carry the bundle even when nothing changed — see passThrough.
	stdoutMode := !dryRun && !inPlace && outputFile == ""

	fmt.Fprintf(os.Stderr, "\nGenerating MQL: %s\n", file)
	fmt.Fprintf(os.Stderr, "  Total queries:   %d\n", len(checks))
	fmt.Fprintf(os.Stderr, "  Need generation: %d\n\n", needGen)
	if needGen == 0 {
		fmt.Fprintln(os.Stderr, "  Nothing to generate.")
		if stdoutMode {
			passThrough(data)
		}
		return 0, 0, nil
	}

	results := gen.Generate(ctx, checks, func(r generate.Result) {
		switch r.Action {
		// everything printed here comes from outside cnspec — the uid from a bundle
		// this run did not author, the explanation and reason from the agent — so
		// it goes through the same normalization that writes text into a bundle.
		// Otherwise an ANSI escape in any of them repaints this progress log.
		case generate.ActionGenerated:
			fmt.Fprintf(os.Stderr, "  ✓ %s: generated\n", bundle.SanitizeText(r.UID))
			if explain && r.Explanation != "" {
				fmt.Fprintf(os.Stderr, "      %s\n", bundle.SanitizeText(r.Explanation))
			}
		case generate.ActionFailed:
			fmt.Fprintf(os.Stderr, "  ✗ %s: %s\n", bundle.SanitizeText(r.UID), bundle.SanitizeText(r.Reason))
		case generate.ActionSkipped:
			// skips are expected and numerous (existing mql, variant parents);
			// the summary reports the count, so stay quiet here.
		}
	})

	// apply generated MQL back onto the exact query definition (targets is
	// index-aligned with checks and results)
	for i, r := range results {
		switch r.Action {
		case generate.ActionGenerated:
			targets[i].Mql = r.MQL
			generated++
		case generate.ActionFailed:
			failed++
		}
	}

	if generated == 0 {
		// nothing to write back, but stdout still owes the caller the bundle
		if stdoutMode {
			passThrough(data)
		}
		return generated, failed, nil
	}

	fmtData, err := bundle.FormatBundle(raw, false)
	if err != nil {
		return generated, failed, fmt.Errorf("could not format %s: %w", file, err)
	}

	switch {
	case dryRun:
		fmt.Fprintf(os.Stderr, "\n--- %s (dry run, %d generated, not written) ---\n", file, generated)
		fmt.Print(string(fmtData))
	case inPlace:
		if err := writeFileAtomic(file, fmtData); err != nil {
			return generated, failed, fmt.Errorf("could not write %s: %w", file, err)
		}
		fmt.Fprintf(os.Stderr, "  Updated: %s\n", file)
	case outputFile != "":
		if err := writeFileAtomic(outputFile, fmtData); err != nil {
			return generated, failed, fmt.Errorf("could not write %s: %w", outputFile, err)
		}
		fmt.Fprintf(os.Stderr, "  Wrote: %s\n", outputFile)
	default:
		fmt.Print(string(fmtData))
	}

	return generated, failed, nil
}

// passThrough copies the input bundle to stdout unchanged.
//
// In stdout mode the shell has already created — and truncated — the destination
// of `cnspec policy generate in.yaml > out.yaml`. Printing nothing when there was
// nothing to generate left a 0-byte file and exit 0, so a scripted
// `generate in.yaml > out.yaml && mv out.yaml in.yaml` destroyed the bundle. The
// original bytes go out rather than a re-format, so a run that changed nothing
// is byte-identical to its input.
func passThrough(data []byte) {
	fmt.Print(string(data))
}

// writeFileAtomic writes data to path via a temp file in the same directory
// followed by a rename, so an interrupted or failed write cannot truncate or
// corrupt the existing policy file. The rename is atomic on the same filesystem.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".gen-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after a successful rename

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	// flush to disk before the rename so a crash cannot leave a zero-length file
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	// preserve the existing file's permissions rather than widening to 0644
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(path); statErr == nil {
		mode = info.Mode().Perm()
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// loadGenerateCorpus picks the grounding corpus: an explicit --corpus path, else
// ./content when present, else the input paths themselves. A failure to load is
// non-fatal — generation still works without grounding, just less accurately.
func loadGenerateCorpus(corpusPath string, inputPaths []string) *generate.Corpus {
	var paths []string
	switch {
	case corpusPath != "":
		paths = []string{corpusPath}
	case dirExists("content"):
		paths = []string{"content"}
	default:
		paths = inputPaths
	}

	corpus, err := generate.LoadCorpus(paths...)
	if err != nil {
		log.Warn().Err(err).Msg("could not load grounding corpus; continuing without examples")
		return nil
	}
	if corpus.Size() == 0 {
		return nil
	}
	fmt.Fprintf(os.Stderr, "Grounding on %d example checks from %s\n", corpus.Size(), strings.Join(paths, ", "))
	return corpus
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// bundleProps converts a query's parameterized properties into the neutral shape
// the generator embeds in its prompt.
func bundleProps(q *bundle.Mquery) []generate.Prop {
	if q == nil || len(q.Props) == 0 {
		return nil
	}
	out := make([]generate.Prop, 0, len(q.Props))
	for _, p := range q.Props {
		if p == nil {
			continue
		}
		// a prop is referenced in MQL by its uid; without one there is no valid
		// props.<name> to hand the agent, so skip it rather than emit garbage.
		name := strings.TrimSpace(p.Uid)
		if name == "" {
			continue
		}
		// authored props carry their human-readable summary in `title`; fall back
		// to it when there is no explicit `desc`.
		desc := strings.TrimSpace(p.Desc)
		if desc == "" {
			desc = strings.TrimSpace(p.Title)
		}
		out = append(out, generate.Prop{
			Name: name,
			Type: strings.TrimSpace(p.Type),
			Desc: desc,
			// most authored props declare no `type:`, so the prop's own query is
			// the only thing that can type `props.<name>` for the compiler.
			Mql: strings.TrimSpace(p.Mql),
		})
	}
	return out
}

// findSkills returns the paths to any discoverable cnspec skill files (the mql
// and policy-graph skills) so the agent can read them. It checks the working
// directory, its parent, and the directory next to the cnspec binary — covering
// both "run from a checkout" and "run beside an installed skills/ dir". Empty
// when none are found, in which case the prompt falls back to a name-based hint.
func findSkills() []string {
	dirs := []string{"skills", filepath.Join("..", "skills")}
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Join(filepath.Dir(exe), "skills"))
	}

	var out []string
	for _, name := range []string{"mql", "policy-graph"} {
		for _, d := range dirs {
			p := filepath.Join(d, name, "SKILL.md")
			if _, err := os.Stat(p); err == nil {
				out = append(out, p)
				break
			}
		}
	}
	return out
}
