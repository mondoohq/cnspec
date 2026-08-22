// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connectorgen

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cockroachdb/errors"
)

// Root is one source tree to read: the mql monorepo, or the enterprise provider
// repository beside it.
type Root struct {
	// Name is the short name recorded in the artifact, e.g. "mql".
	Name string
	// Path is the checkout to read.
	Path string
}

// Extract reads every provider under every root and returns the artifact.
//
// It refuses rather than degrades when a root is not a provider checkout. The
// artifact is checked in, so a run that quietly produced half of it would
// replace a complete file with an incomplete one and nothing downstream would
// notice; a run that stops leaves the checked-in file alone.
func Extract(roots []Root) (*Artifact, error) {
	if len(roots) == 0 {
		return nil, errors.New("no source tree given: point the generator at an mql checkout")
	}

	art := &Artifact{
		Schema:      SchemaVersion,
		GeneratedBy: "go.mondoo.com/cnspec/internal/connectorgen",
	}
	seen := map[string]string{}

	for _, root := range roots {
		providersDir := filepath.Join(root.Path, "providers")
		info, err := os.Stat(providersDir)
		if err != nil || !info.IsDir() {
			return nil, errors.Newf("%s is not a provider checkout: no %s directory", root.Path, providersDir)
		}

		src := Source{Name: root.Name}
		src.Commit, src.Dirty = gitState(root.Path)

		found, gaps, err := extractRoot(root, providersDir)
		if err != nil {
			return nil, err
		}
		src.Providers = len(found)
		art.Sources = append(art.Sources, src)
		art.Gaps = append(art.Gaps, gaps...)

		for _, c := range found {
			key := c.Provider + "/" + c.Name
			if from, dup := seen[key]; dup {
				art.Gaps = append(art.Gaps, Gap{
					Provider:  c.Provider,
					Connector: c.Name,
					Kind:      GapConfigNotLiteral,
					Detail:    fmt.Sprintf("declared in both %s and %s; the first was kept", from, root.Name),
				})
				continue
			}
			seen[key] = root.Name
			art.Connectors = append(art.Connectors, c)
		}
	}

	if len(art.Connectors) == 0 {
		return nil, errors.New("no connectors found: the checkout has providers/ but no plugin.Provider literal in it")
	}

	sort.SliceStable(art.Connectors, func(i, j int) bool {
		if art.Connectors[i].Name != art.Connectors[j].Name {
			return art.Connectors[i].Name < art.Connectors[j].Name
		}
		return art.Connectors[i].Provider < art.Connectors[j].Provider
	})
	sortGaps(art.Gaps)

	// Each connector carries its own share of the gaps, so a consumer reading
	// one connector sees what is missing from it without joining two lists.
	byConnector := map[string][]Gap{}
	for _, g := range art.Gaps {
		if g.Connector == "" {
			continue
		}
		byConnector[g.Provider+"/"+g.Connector] = append(byConnector[g.Provider+"/"+g.Connector], g)
	}
	for i := range art.Connectors {
		c := &art.Connectors[i]
		for _, g := range byConnector[c.Provider+"/"+c.Name] {
			// Repeated inside the connector, so provider and connector would
			// only say twice what the position in the file already says.
			g.Provider, g.Connector = "", ""
			c.Gaps = append(c.Gaps, g)
		}
	}
	return art, nil
}

// extractRoot walks one checkout's providers directory.
func extractRoot(root Root, providersDir string) ([]Connector, []Gap, error) {
	entries, err := os.ReadDir(providersDir)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "cannot read %s", providersDir)
	}

	var connectors []Connector
	var gaps []Gap

	// The providers that live in their own directory. Their name comes from
	// the literal rather than the directory: providers/claude declares itself
	// "claude", and a walk keying on the directory would mislabel any provider
	// whose two names differ.
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(providersDir, e.Name())
		cfgDir := filepath.Join(dir, "config")
		if _, err := os.Stat(filepath.Join(cfgDir, "config.go")); err != nil {
			// A directory holding a built artifact or a resource schema but no
			// config is a provider whose source is not in this checkout --
			// equinix is one here. Saying so is the difference between a
			// connector that is missing for a reason and one that is missing
			// silently.
			if looksLikeProvider(dir) {
				gaps = append(gaps, Gap{
					Provider: e.Name(),
					Kind:     GapSourceAbsent,
					Detail:   "the directory holds a built provider but no config package, so nothing about it could be read",
					Where:    relPath(root, dir),
				})
			}
			continue
		}
		cs, gs, err := extractProvider(root, dir, cfgDir, filepath.Join(dir, "provider"))
		if err != nil {
			return nil, nil, err
		}
		connectors = append(connectors, cs...)
		gaps = append(gaps, gs...)
	}

	// The built-in providers, declared in files directly under providers/
	// rather than in a provider directory. sbom is one, and it is a connector a
	// user can select, so leaving it out would make the artifact quietly
	// smaller than the launcher's list.
	cs, gs, err := extractBuiltins(root, providersDir)
	if err != nil {
		return nil, nil, err
	}
	connectors = append(connectors, cs...)
	gaps = append(gaps, gs...)

	return connectors, gaps, nil
}

// extractProvider reads one providers/<dir> tree.
func extractProvider(root Root, dir, cfgDir, provDir string) ([]Connector, []Gap, error) {
	fset := token.NewFileSet()
	sy := newSymbols(fset)
	// The whole provider tree, because the config literal names constants that
	// live in the connection and resources packages beside it.
	if err := sy.scanTree(dir); err != nil {
		return nil, nil, err
	}
	where := positionFunc(fset, root)

	declared, ok := parseConfig(sy, sy.files[cfgDir], where)
	if !ok {
		return nil, nil, nil
	}
	if declared.Name == "" {
		declared.Name = filepath.Base(dir)
	}
	// Every package under connection/, not just the top one: atlassian splits
	// its two products into connection/jira and connection/confluence, and each
	// reads its own token.
	return assemble(sy, declared, sy.files[provDir], sy.filesUnder(filepath.Join(dir, "connection")),
		where, relPath(root, provDir))
}

// extractBuiltins reads the provider literals declared directly under
// providers/, which have no config package and no provider package of their
// own: ParseCLI for those sits in the same package.
func extractBuiltins(root Root, providersDir string) ([]Connector, []Gap, error) {
	fset := token.NewFileSet()
	sy := newSymbols(fset)
	entries, err := os.ReadDir(providersDir)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "cannot read %s", providersDir)
	}
	var files []*ast.File
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, perr := parseOne(fset, filepath.Join(providersDir, e.Name()))
		if perr != nil {
			continue
		}
		files = append(files, f)
		sy.record(f)
	}
	if len(files) == 0 {
		return nil, nil, nil
	}
	where := positionFunc(fset, root)

	var connectors []Connector
	var gaps []Gap
	for _, f := range files {
		declared, ok := parseConfig(sy, []*ast.File{f}, where)
		if !ok || declared.Name == "" {
			continue
		}
		// providers/defaults.go is an auto-generated list of every provider
		// with the flags, argument counts and discovery targets stripped -- the
		// air-gapped fallback. It is the very thing the launcher's catalog
		// degrades to when nothing is installed, so reading it as a source of
		// truth would replace 81 real declarations with 81 empty ones.
		if !declaresMetadata(declared.Connectors) {
			continue
		}
		// The provider package is the one file, not everything under
		// providers/: the coordinator lives there too, and reading its
		// environment lookups as though they belonged to a connector produced
		// four providers that all claimed to read PROVIDERS_PATH.
		cs, gs, err := assemble(sy, declared, []*ast.File{f}, nil, where, relPath(root, providersDir))
		if err != nil {
			return nil, nil, err
		}
		connectors = append(connectors, cs...)
		gaps = append(gaps, gs...)
	}
	return connectors, gaps, nil
}

// assemble joins what the config declared to what the provider package does
// with it, and records everything the join could not settle.
func assemble(sy *symbols, declared declaredProvider, provFiles, connFiles []*ast.File, where func(ast.Node) string, provDir string) ([]Connector, []Gap, error) {
	gaps := declared.Gaps

	var analysis providerAnalysis
	switch {
	case len(provFiles) == 0:
		gaps = append(gaps, Gap{
			Provider: declared.Name,
			Kind:     GapNoProviderPkg,
			Detail:   "no provider package beside the config, so no ParseCLI to read environment variables or sub-commands out of",
			Where:    provDir,
		})
	default:
		analysis = analyzeProviderPkg(sy, provFiles, where, flagKeys)
		if !hasFunc(provFiles, "ParseCLI") {
			gaps = append(gaps, Gap{
				Provider: declared.Name,
				Kind:     GapNoParseCLI,
				Detail:   "the provider package declares no ParseCLI",
				Where:    provDir,
			})
		}
	}

	// The connection package is the second place a credential arrives from, and
	// for roughly a fifth of the tree it is the only place: zoom, tailscale and
	// openstack read their tokens at connect time out of the options map rather
	// than in ParseCLI. The option names there are not flag names in general,
	// so only the ones a connector actually declares are kept; the rest are
	// reported.
	var connAnalysis providerAnalysis
	if len(connFiles) > 0 {
		connAnalysis = analyzeProviderPkg(sy, connFiles, where, optionKeys)
	}

	for i := range analysis.Gaps {
		analysis.Gaps[i].Provider = declared.Name
	}
	gaps = append(gaps, analysis.Gaps...)

	// Which connector does an association belong to? The one that declares the
	// flag. A provider shipping several connectors -- os ships eight -- has one
	// ParseCLI serving all of them, so the flag name is the only thing in the
	// syntax that distinguishes them, and it is a fact rather than a guess.
	declaresFlag := func(c Connector, flag string) bool {
		for _, fl := range c.Flags {
			if fl.Long == flag {
				return true
			}
		}
		return false
	}

	out := make([]Connector, 0, len(declared.Connectors))
	for _, c := range declared.Connectors {
		c.Provider = declared.Name
		for _, fe := range analysis.Env {
			if !declaresFlag(c, fe.Flag) {
				continue
			}
			fe.Declared = true
			c.Env = append(c.Env, fe)
		}
		for _, fe := range connAnalysis.Env {
			if !declaresFlag(c, fe.Flag) || hasFlagEnv(c.Env, fe.Flag) {
				continue
			}
			fe.Declared = true
			c.Env = append(c.Env, fe)
		}
		sort.SliceStable(c.Env, func(i, j int) bool { return c.Env[i].Flag < c.Env[j].Flag })
		out = append(out, c)
	}

	// What the connection walk found and nothing could use. Reported as one
	// finding per provider: a variable read there may back an option that is
	// not a flag, or a value the CLI has no way to pass at all, and either way
	// the launcher cannot offer a field for it.
	var loose []string
	for _, fe := range connAnalysis.Env {
		used := false
		for _, c := range out {
			if declaresFlag(c, fe.Flag) {
				used = true
				break
			}
		}
		if !used {
			loose = appendUnique(loose, fmt.Sprintf("%s (option %q)", strings.Join(fe.Vars, ", "), fe.Flag))
		}
	}
	for _, name := range connAnalysis.Unbound {
		// A variable the connection package also reads while parsing is not
		// loose; it already has its route, and repeating it here would make the
		// gap list longer than the thing it is reporting on.
		if attachedAnywhere(out, name) {
			continue
		}
		loose = appendUnique(loose, name)
	}
	if len(loose) > 0 {
		gaps = append(gaps, Gap{
			Provider: declared.Name,
			Kind:     GapEnvOutsideParseCLI,
			Detail: fmt.Sprintf("the connection package reads %s, which no declared flag accounts for",
				strings.Join(loose, "; ")),
		})
	}

	for _, fe := range analysis.Env {
		attached := false
		for _, c := range out {
			if declaresFlag(c, fe.Flag) {
				attached = true
				break
			}
		}
		if attached {
			continue
		}
		// A value the provider reads from the environment for a flag no
		// connector declares. Either the flag was removed and the read was
		// left behind, or the value has no command-line route at all -- which
		// is exactly the case a launcher needs told, because it cannot offer a
		// field for it.
		gaps = append(gaps, Gap{
			Provider: declared.Name,
			Kind:     GapUndeclaredFlag,
			Detail: fmt.Sprintf("%s reads %q from %s but no connector declares that flag",
				fe.Func, fe.Flag, strings.Join(fe.Vars, ", ")),
		})
	}

	gaps = append(gaps, attributePositional(&out, declared.Name, analysis.Positional)...)

	return out, gaps, nil
}

// attributePositional hands the sub-command vocabulary to the connector it
// belongs to, and says so when it cannot tell which one that is.
func attributePositional(connectors *[]Connector, provider string, found []Positional) []Gap {
	var gaps []Gap
	list := *connectors

	// One ParseCLI serves every connector a provider ships, so a vocabulary
	// found there belongs to whichever of them takes positional arguments. When
	// exactly one does, that is a fact rather than a guess: the others cannot
	// receive an argument at all.
	var takesArgs []int
	for i := range list {
		if list[i].MaxArgs > 0 {
			takesArgs = append(takesArgs, i)
		}
	}
	if len(takesArgs) == 1 {
		list[takesArgs[0]].Positional = found
	} else if len(takesArgs) == 0 && len(found) > 0 {
		gaps = append(gaps, Gap{
			Provider: provider,
			Kind:     GapComputedPositional,
			Detail:   "ParseCLI compares positional arguments that no connector declares it takes",
		})
	} else if len(found) > 0 {
		words := make([]string, 0, len(found))
		for _, p := range found {
			words = append(words, fmt.Sprintf("argument %d: %s", p.Index, strings.Join(p.Values, "|")))
		}
		gaps = append(gaps, Gap{
			Provider: provider,
			Kind:     GapPositionalAmbiguous,
			Detail: fmt.Sprintf("one ParseCLI serves %d connectors that take positional arguments, so %s cannot be attributed to one of them",
				len(takesArgs), strings.Join(words, "; ")),
		})
	}

	// A connector taking two or more arguments is the sub-command shape: the
	// first word selects and the rest are its value. `mongo <host>` takes one
	// argument and has no vocabulary to find, so only the two-argument shape is
	// reported when nothing was found -- otherwise every host connector in the
	// catalog would file a gap saying its hostname is not a keyword.
	for i := range list {
		if list[i].MinArgs >= 2 && len(list[i].Positional) == 0 {
			gaps = append(gaps, Gap{
				Provider:  provider,
				Connector: list[i].Name,
				Kind:      GapNoPositionalVocabulary,
				Detail: fmt.Sprintf("takes %d arguments but no literal comparison against them was found, so the words it accepts are documented only in its help text",
					list[i].MinArgs),
			})
		}
	}
	return gaps
}

// looksLikeProvider reports whether a directory under providers/ is a provider
// whose source is missing, rather than something else that happens to live
// there.
func looksLikeProvider(dir string) bool {
	for _, marker := range []string{"dist", "resources"} {
		if info, err := os.Stat(filepath.Join(dir, marker)); err == nil && info.IsDir() {
			return true
		}
	}
	return false
}

// declaresMetadata reports whether any connector carries its own declaration,
// as opposed to being a bare name in a fallback list. It is the same question
// the launcher's own catalog asks of an installed provider.
func declaresMetadata(connectors []Connector) bool {
	for _, c := range connectors {
		if len(c.Flags) > 0 || c.MaxArgs > 0 || len(c.Discovery) > 0 {
			return true
		}
	}
	return false
}

func hasFunc(files []*ast.File, name string) bool {
	for _, f := range files {
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && fd.Name.Name == name {
				return true
			}
		}
	}
	return false
}

// positionFunc renders a node's position relative to the checkout, prefixed
// with the repository name so a gap from the enterprise tree is not mistaken
// for one from mql.
func positionFunc(fset *token.FileSet, root Root) func(ast.Node) string {
	return func(n ast.Node) string {
		if n == nil {
			return ""
		}
		p := fset.Position(n.Pos())
		rel, err := filepath.Rel(root.Path, p.Filename)
		if err != nil {
			rel = p.Filename
		}
		return fmt.Sprintf("%s:%s:%d", root.Name, filepath.ToSlash(rel), p.Line)
	}
}

func relPath(root Root, path string) string {
	rel, err := filepath.Rel(root.Path, path)
	if err != nil {
		return path
	}
	return root.Name + ":" + filepath.ToSlash(rel)
}

func sortGaps(gaps []Gap) {
	sort.SliceStable(gaps, func(i, j int) bool {
		a, b := gaps[i], gaps[j]
		if a.Provider != b.Provider {
			return a.Provider < b.Provider
		}
		if a.Connector != b.Connector {
			return a.Connector < b.Connector
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Detail != b.Detail {
			return a.Detail < b.Detail
		}
		return a.Where < b.Where
	})
}

// gitState reports the checkout's revision, so the artifact says what it was
// generated from. A tree that is not a git repository is not an error: the
// commit is provenance, not input.
func gitState(path string) (commit string, dirty bool) {
	out, err := exec.Command("git", "-C", path, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", false
	}
	commit = strings.TrimSpace(string(out))
	// Scoped to providers/, because that is the only part of the tree this
	// reads. It still counts untracked files there, some of which the walk
	// skips: overstating the dirtiness of a provenance field costs a reader a
	// second look, understating it costs them a wrong answer.
	status, err := exec.Command("git", "-C", path, "status", "--porcelain", "--", "providers").Output()
	if err != nil {
		return commit, false
	}
	return commit, len(strings.TrimSpace(string(status))) > 0
}

// attachedAnywhere reports whether an environment variable already backs a flag
// on some connector of this provider.
func attachedAnywhere(connectors []Connector, envVar string) bool {
	for _, c := range connectors {
		for _, fe := range c.Env {
			if contains(fe.Vars, envVar) {
				return true
			}
		}
	}
	return false
}

// hasFlagEnv reports whether a flag already has an association, so the
// connection walk does not restate what ParseCLI already settled.
func hasFlagEnv(list []FlagEnv, flag string) bool {
	for _, fe := range list {
		if fe.Flag == flag {
			return true
		}
	}
	return false
}
