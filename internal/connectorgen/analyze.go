// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connectorgen

import (
	"fmt"
	"go/ast"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// Reading providers/<name>/provider/*.go for the two facts nothing else
// carries: which environment variable backs which flag, and which literal words
// a positional argument is compared against.
//
// Both are ordinary Go control flow, and the shape is the same everywhere:
// a local variable is filled from the flag map, and if it is still empty it is
// filled from the environment.
//
//	token := ""
//	if x, ok := flags["token"]; ok && len(x.Value) != 0 {
//		token = string(x.Value)
//	}
//	if token == "" && len(os.Getenv("GITHUB_TOKEN")) != 0 {
//		token = os.Getenv("GITHUB_TOKEN")
//	}
//
// So the association is transitive through local variables: `x` holds
// flags["token"], `token` holds `x`, `token` holds GITHUB_TOKEN, therefore
// --token travels in GITHUB_TOKEN. What the walk below does is propagate flag
// names and variable names along assignments until they meet.
//
// The trap is that `x` is a name providers reuse. github binds it to
// flags["token"] in one if statement and to the app private key flag in
// another, and a walk that keyed variables by name alone would conclude that
// --token travels in GITHUB_APP_PRIVATE_KEY. Variables are therefore keyed by
// their declaration position and resolved through a scope stack, which is why
// this is an explicit statement walker rather than an ast.Inspect.

// providerAnalysis is what one provider package yielded.
type providerAnalysis struct {
	Env        []FlagEnv
	Positional []Positional
	Gaps       []Gap
	// Unbound are environment variables read but tied to no key, collected
	// rather than reported one at a time. Only the connection walk fills it.
	Unbound []string
}

// entry points whose reachability makes an environment read attributable. A
// variable read anywhere they lead to is read while parsing a command line;
// one read elsewhere is read at some other time, by something the CLI has no
// say over, and this tool will not claim to know which flag it belongs to.
var analysisRoots = []string{"ParseCLI", "Connect", "MockConnect"}

// analyzeProviderPkg walks a provider package.
//
// It walks it twice. The first pass learns what each package-local function
// returns -- mistral's envToken() returns MISTRAL_API_KEY and takes no argument
// that says so -- and the second pass uses those summaries at the call sites,
// which is what lets a variable holding --token meet a variable name declared
// two functions away.
func analyzeProviderPkg(sy *symbols, files []*ast.File, where func(ast.Node) string, key keySource) providerAnalysis {
	helpers := findHelpers(files, key)

	var decls []*ast.FuncDecl
	byName := map[string][]*ast.FuncDecl{}
	for _, f := range files {
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			decls = append(decls, fd)
			byName[fd.Name.Name] = append(byName[fd.Name.Name], fd)
		}
	}
	// In a provider package the command-line parser is the entry point and
	// anything it does not reach is read at some other time. A connection
	// package has no such entry point -- every exported function there runs
	// while connecting -- so all of it counts.
	reachable := map[string]bool{}
	if key.reportUnbound {
		reachable = reachableFuncs(byName, analysisRoots)
	} else {
		for name := range byName {
			reachable[name] = true
		}
	}

	// A stable walk order, so a regeneration diff is about the source rather
	// than about map iteration.
	sort.SliceStable(decls, func(i, j int) bool { return decls[i].Pos() < decls[j].Pos() })

	argsParams := argumentSliceParams(byName)

	newAnalyzer := func(fd *ast.FuncDecl, summaries map[string]funcSummary) *funcAnalyzer {
		args := map[string]bool{}
		for name := range argsParams[fd.Name.Name] {
			args[name] = true
		}
		return &funcAnalyzer{
			sy:          sy,
			helpers:     helpers,
			summaries:   summaries,
			key:         key,
			fn:          fd.Name.Name,
			where:       where,
			vars:        map[string]*varInfo{},
			flagsIdents: map[string]bool{},
			argsIdents:  args,
		}
	}

	summaries := map[string]funcSummary{}
	for _, fd := range decls {
		if !stringReturning(fd.Type.Results) {
			continue
		}
		a := newAnalyzer(fd, nil)
		a.run(fd)
		if s := a.summary(); len(s.flags) > 0 || len(s.envs) > 0 {
			summaries[fd.Name.Name] = s
		}
	}

	var out providerAnalysis
	envByFlag := map[string]*FlagEnv{}
	var flagOrder []string
	posByIndex := map[int][]string{}
	var posFunc = map[int]string{}
	boundEnvs := map[string]bool{}

	for _, fd := range decls {
		a := newAnalyzer(fd, summaries)
		a.run(fd)

		if !reachable[fd.Name.Name] {
			// An environment read the command-line parser never reaches. The
			// variable is real, the flag it belongs to is not knowable here.
			if len(a.envsSeen) > 0 {
				out.Gaps = append(out.Gaps, Gap{
					Kind: GapEnvOutsideParseCLI,
					Detail: fmt.Sprintf("%s reads %s but is not reached from %s, so no flag can be attributed",
						fd.Name.Name, strings.Join(a.envsSeen, ", "), strings.Join(analysisRoots, "/")),
					Where: where(fd),
				})
			}
			out.Gaps = append(out.Gaps, a.gaps...)
			continue
		}

		// results() is what resolves the walk, and resolving is what raises the
		// ambiguous-binding and alternative-branch gaps, so a.gaps is read
		// after it rather than before.
		resolved := a.results()
		for _, fe := range resolved {
			for _, v := range fe.Vars {
				boundEnvs[v] = true
			}
			existing, ok := envByFlag[fe.Flag]
			if !ok {
				copied := fe
				envByFlag[fe.Flag] = &copied
				flagOrder = append(flagOrder, fe.Flag)
				continue
			}
			for _, v := range fe.Vars {
				existing.Vars = appendUnique(existing.Vars, v)
			}
			existing.Composed = existing.Composed || fe.Composed
		}
		for _, idx := range a.posOrder {
			for _, v := range a.positional[idx] {
				posByIndex[idx] = appendUnique(posByIndex[idx], v)
			}
			if _, seen := posFunc[idx]; !seen {
				posFunc[idx] = fd.Name.Name
			}
		}
		// An environment variable read while parsing that never met a flag.
		for _, name := range a.envsSeen {
			if boundEnvs[name] {
				continue
			}
			if a.boundLater(name) {
				continue
			}
			// A function that returns the value is not losing it; its caller
			// is where the pairing happens, and the caller is walked too.
			if _, summarised := summaries[fd.Name.Name]; summarised && contains(a.summary().envs, name) {
				continue
			}
			if !key.reportUnbound {
				// The caller reports the connection package's loose variables
				// as one finding rather than one per function.
				out.Unbound = appendUnique(out.Unbound, name)
				continue
			}
			out.Gaps = append(out.Gaps, Gap{
				Kind:   GapUnboundEnv,
				Detail: fmt.Sprintf("%s reads %s but its value never meets a flag", fd.Name.Name, name),
				Where:  where(fd),
			})
		}
		out.Gaps = append(out.Gaps, a.gaps...)
	}

	for _, flag := range flagOrder {
		out.Env = append(out.Env, *envByFlag[flag])
	}
	sort.SliceStable(out.Env, func(i, j int) bool { return out.Env[i].Flag < out.Env[j].Flag })

	var indexes []int
	for idx := range posByIndex {
		indexes = append(indexes, idx)
	}
	sort.Ints(indexes)
	for _, idx := range indexes {
		out.Positional = append(out.Positional, Positional{
			Index:  idx,
			Values: posByIndex[idx],
			Func:   posFunc[idx],
		})
	}
	return out
}

// reachableFuncs is the set of package-local function names the roots lead to.
//
// It is a name-level call graph, which is exactly as much as syntax supports:
// a call through an interface or a stored function value is invisible here.
// That imprecision only ever widens the set, and a widened set means an
// association is recorded rather than dropped, so the tool's mistakes land on
// the side of reporting rather than of silence.
func reachableFuncs(byName map[string][]*ast.FuncDecl, roots []string) map[string]bool {
	seen := map[string]bool{}
	queue := append([]string(nil), roots...)
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if seen[name] {
			continue
		}
		seen[name] = true
		for _, fd := range byName[name] {
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				switch fn := call.Fun.(type) {
				case *ast.Ident:
					if _, local := byName[fn.Name]; local && !seen[fn.Name] {
						queue = append(queue, fn.Name)
					}
				case *ast.SelectorExpr:
					// A method on the service, s.connect(...), is local too.
					if _, local := byName[fn.Sel.Name]; local && !seen[fn.Sel.Name] {
						queue = append(queue, fn.Sel.Name)
					}
				}
				return true
			})
		}
	}
	return seen
}

// argumentSliceParams finds the parameters that receive the positional argument
// slice, per package-local function.
//
// aws is why this exists. Its ParseCLI compares req.Args[0] against "ec2" and
// then hands the whole slice to handleAwsEc2Subcommands, which compares the
// next word against the rest of the vocabulary. Without following the slice
// into that function the artifact would record aws as accepting exactly one
// sub-command -- a complete-looking answer that is wrong, which is the one
// outcome this tool is built to avoid.
//
// It iterates because the slice is passed on again: ParseCLI to one function,
// that function to another.
func argumentSliceParams(byName map[string][]*ast.FuncDecl) map[string]map[string]bool {
	out := map[string]map[string]bool{}

	isArgs := func(fn string, e ast.Expr) bool {
		if id, ok := e.(*ast.Ident); ok {
			return out[fn][id.Name]
		}
		return readsField(e, "Args")
	}

	for range len(byName) + 1 {
		changed := false
		for name, decls := range byName {
			for _, fd := range decls {
				ast.Inspect(fd.Body, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					var callee string
					switch f := call.Fun.(type) {
					case *ast.Ident:
						callee = f.Name
					case *ast.SelectorExpr:
						callee = f.Sel.Name
					default:
						return true
					}
					target, local := byName[callee]
					if !local || len(target) == 0 {
						return true
					}
					params := flatParams(target[0].Type.Params)
					for i, arg := range call.Args {
						if i >= len(params) || params[i].name == "" {
							continue
						}
						if !isArgs(name, arg) {
							continue
						}
						if out[callee] == nil {
							out[callee] = map[string]bool{}
						}
						if !out[callee][params[i].name] {
							out[callee][params[i].name] = true
							changed = true
						}
					}
					return true
				})
			}
		}
		if !changed {
			break
		}
	}
	return out
}

// factGroup is what one assignment's right-hand side contributed.
type factGroup struct {
	flags []string
	envs  []string
	refs  []string
	// selected counts the environment variables this expression chose between
	// rather than joined. An accessor called as stringFlag(flags, "region",
	// "ALIBABA_CLOUD_REGION", "ALIBABA_CLOUD_REGION_ID") returns the first that
	// is set, so two variables there are a fallback chain, while two variables
	// concatenated by a "+" are a composition. The two need opposite advice --
	// set either one, or set both -- so they are counted apart.
	selected int
}

// branchChain records one name declared by more than one arm of a single
// if/else chain.
type branchChain struct {
	name string
	keys []string
}

// varInfo accumulates what a single local variable came to hold.
type varInfo struct {
	name   string
	groups []factGroup
	flags  []string
	envs   []string
}

// funcAnalyzer walks one function body.
type funcAnalyzer struct {
	sy        *symbols
	helpers   map[string]helperSig
	summaries map[string]funcSummary
	key       keySource
	fn        string
	where     func(ast.Node) string

	scopes      []map[string]string
	vars        map[string]*varInfo
	order       []string
	flagsIdents map[string]bool
	argsIdents  map[string]bool

	// direct are associations read straight off a helper call, which need no
	// variable to travel through.
	direct []FlagEnv
	// envsSeen is every environment variable this function reads, in order.
	envsSeen []string

	positional map[int][]string
	posOrder   []int

	gaps []Gap

	// branchChains are the names one if/else chain declared in more than one
	// arm, which computeResults reads for the alternative-branch gap.
	branchChains []branchChain
	// explained are environment variables a gap already accounts for, so the
	// broad "never meets a flag" report does not repeat a finding the specific
	// one already made.
	explained []string

	// returns are the facts flowing to this function's own return statements,
	// which is what summary() reports to callers. Only the outermost function's
	// returns count, so closures do not contribute.
	returns  []factGroup
	litDepth int

	// resolved and cached memoise results(): resolving raises gaps, and a
	// second call would raise every one of them twice.
	resolved bool
	cached   []FlagEnv
}

func (a *funcAnalyzer) run(fd *ast.FuncDecl) {
	a.positional = map[int][]string{}
	a.push()
	a.declareParams(fd.Type.Params)
	for _, st := range fd.Body.List {
		a.stmt(st)
	}
	a.pop()
	a.fixpoint()
}

// declareParams binds a function's parameters, so a flag name that is a
// parameter reads as a value the call site supplies rather than as a constant
// that failed to resolve.
func (a *funcAnalyzer) declareParams(fl *ast.FieldList) {
	for _, p := range flatParams(fl) {
		if p.name == "" || p.name == "_" {
			continue
		}
		key := p.name + "#param"
		if len(a.scopes) > 0 {
			a.scopes[len(a.scopes)-1][p.name] = key
		}
		if _, ok := a.vars[key]; !ok {
			a.vars[key] = &varInfo{name: p.name}
			a.order = append(a.order, key)
		}
		if a.key.typeOK != nil && a.key.typeOK(p.typ) {
			a.flagsIdents[p.name] = true
		}
	}
}

// summary is what a caller learns by calling this function: the flag names and
// environment variables that flow into what it returns.
func (a *funcAnalyzer) summary() funcSummary {
	var s funcSummary
	for _, g := range a.returns {
		addAll(&s.flags, g.flags)
		addAll(&s.envs, g.envs)
		for _, ref := range g.refs {
			if v, ok := a.vars[ref]; ok {
				addAll(&s.flags, v.flags)
				addAll(&s.envs, v.envs)
			}
		}
	}
	return s
}

// localBinding reports whether an expression is an identifier bound inside this
// function -- a parameter, a local, a range variable.
//
// It is what separates a flag name the syntax genuinely does not know from one
// the call site supplies. `flags[name]` inside an accessor is not a gap: the
// name is an argument, and the argument is read where the accessor is called.
// `flags[someConstant]` that failed to resolve is a gap, because nothing else
// will supply it.
func (a *funcAnalyzer) localBinding(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	if !ok {
		return false
	}
	return a.resolve(id.Name) != id.Name
}

func (a *funcAnalyzer) gap(kind, detail string, n ast.Node) {
	a.gaps = append(a.gaps, Gap{Kind: kind, Detail: detail, Where: a.where(n)})
}

// --- scopes -------------------------------------------------------------

func (a *funcAnalyzer) push() { a.scopes = append(a.scopes, map[string]string{}) }
func (a *funcAnalyzer) pop()  { a.scopes = a.scopes[:len(a.scopes)-1] }

// declare binds a name to a fresh key for the innermost scope. The key carries
// the declaration offset, which is what keeps github's two `x` variables apart.
func (a *funcAnalyzer) declare(id *ast.Ident) string {
	key := id.Name + "#" + strconv.Itoa(int(id.Pos()))
	if len(a.scopes) > 0 {
		a.scopes[len(a.scopes)-1][id.Name] = key
	}
	if _, ok := a.vars[key]; !ok {
		a.vars[key] = &varInfo{name: id.Name}
		a.order = append(a.order, key)
	}
	return key
}

// resolve finds the innermost binding of a name, or returns the bare name for
// something declared outside this function.
func (a *funcAnalyzer) resolve(name string) string {
	for i := len(a.scopes) - 1; i >= 0; i-- {
		if key, ok := a.scopes[i][name]; ok {
			return key
		}
	}
	return name
}

// --- statements ---------------------------------------------------------

func (a *funcAnalyzer) stmt(s ast.Stmt) {
	switch x := s.(type) {
	case nil:
		return
	case *ast.BlockStmt:
		a.push()
		for _, st := range x.List {
			a.stmt(st)
		}
		a.pop()
	case *ast.IfStmt:
		a.ifChain(x)
	case *ast.ForStmt:
		a.push()
		a.stmt(x.Init)
		a.expr(x.Cond)
		a.stmt(x.Post)
		a.stmt(x.Body)
		a.pop()
	case *ast.RangeStmt:
		a.push()
		if x.Tok == token.DEFINE {
			for _, e := range []ast.Expr{x.Key, x.Value} {
				if id, ok := e.(*ast.Ident); ok && id.Name != "_" {
					a.declare(id)
				}
			}
		}
		a.expr(x.X)
		a.stmt(x.Body)
		a.pop()
	case *ast.SwitchStmt:
		a.push()
		a.stmt(x.Init)
		a.switchStmt(x)
		a.pop()
	case *ast.TypeSwitchStmt:
		a.push()
		a.stmt(x.Init)
		a.stmt(x.Assign)
		a.stmt(x.Body)
		a.pop()
	case *ast.SelectStmt:
		a.stmt(x.Body)
	case *ast.CaseClause:
		a.push()
		for _, e := range x.List {
			a.expr(e)
		}
		for _, st := range x.Body {
			a.stmt(st)
		}
		a.pop()
	case *ast.CommClause:
		a.push()
		a.stmt(x.Comm)
		for _, st := range x.Body {
			a.stmt(st)
		}
		a.pop()
	case *ast.LabeledStmt:
		a.stmt(x.Stmt)
	case *ast.AssignStmt:
		a.assign(x)
	case *ast.DeclStmt:
		a.declStmt(x)
	case *ast.ExprStmt:
		a.expr(x.X)
	case *ast.ReturnStmt:
		for _, e := range x.Results {
			g := a.facts(e)
			if a.litDepth == 0 {
				a.returns = append(a.returns, g)
			}
		}
	case *ast.GoStmt:
		a.expr(x.Call)
	case *ast.DeferStmt:
		a.expr(x.Call)
	case *ast.IncDecStmt:
		a.expr(x.X)
	case *ast.SendStmt:
		a.expr(x.Chan)
		a.expr(x.Value)
	default:
		// Anything else contributes no binding; walk it for the side effects
		// facts() records -- an environment read, a positional comparison.
		ast.Inspect(s, func(n ast.Node) bool {
			if e, ok := n.(ast.Expr); ok {
				a.expr(e)
				return false
			}
			return true
		})
	}
}

// ifChain walks an if / else-if / else chain, and while doing so notes the
// names each arm declares in its own init statement.
//
// Three AI providers write their credential this way:
//
//	if token := stringFlag(flags, connection.OptionToken); token != "" {
//		conf.Credentials = append(...)
//	} else if token := envToken(); token != "" {
//		conf.Credentials = append(...)
//	}
//
// Those are two different variables that happen to share a name. The flag and
// the variable never meet in one value, so nothing in the syntax says they are
// the same credential -- only the author's choice of the same name does, and a
// name is not a fact. The chain is recorded here and reported as a gap in
// computeResults rather than resolved into an association.
func (a *funcAnalyzer) ifChain(root *ast.IfStmt) {
	byName := map[string][]string{}
	var order []string

	var walk func(s *ast.IfStmt)
	walk = func(s *ast.IfStmt) {
		a.push()
		if init, ok := s.Init.(*ast.AssignStmt); ok && init.Tok == token.DEFINE {
			a.assign(init)
			for _, lhs := range init.Lhs {
				id, ok := lhs.(*ast.Ident)
				if !ok || id.Name == "_" {
					continue
				}
				if _, seen := byName[id.Name]; !seen {
					order = append(order, id.Name)
				}
				byName[id.Name] = appendUnique(byName[id.Name], a.resolve(id.Name))
			}
		} else {
			a.stmt(s.Init)
		}
		a.expr(s.Cond)
		a.stmt(s.Body)
		switch next := s.Else.(type) {
		case *ast.IfStmt:
			walk(next)
		case nil:
		default:
			a.stmt(s.Else)
		}
		a.pop()
	}
	walk(root)

	for _, name := range order {
		if len(byName[name]) > 1 {
			a.branchChains = append(a.branchChains, branchChain{name: name, keys: byName[name]})
		}
	}
}

func (a *funcAnalyzer) switchStmt(sw *ast.SwitchStmt) {
	idx, isArgs := -1, false
	if sw.Tag != nil {
		idx, isArgs = a.argsIndex(sw.Tag)
		if !isArgs {
			a.expr(sw.Tag)
		}
	}
	for _, st := range sw.Body.List {
		cc, ok := st.(*ast.CaseClause)
		if !ok {
			a.stmt(st)
			continue
		}
		for _, e := range cc.List {
			if !isArgs {
				a.expr(e)
				continue
			}
			v, ok := a.sy.str(e)
			if !ok {
				a.gap(GapComputedPositional,
					fmt.Sprintf("%s switches on positional argument %d against something other than a literal", a.fn, idx), e)
				continue
			}
			if v != "" {
				a.recordPositional(idx, v)
			}
		}
		a.push()
		for _, s := range cc.Body {
			a.stmt(s)
		}
		a.pop()
	}
}

func (a *funcAnalyzer) declStmt(d *ast.DeclStmt) {
	gd, ok := d.Decl.(*ast.GenDecl)
	if !ok || gd.Tok != token.VAR {
		return
	}
	for _, spec := range gd.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for i, name := range vs.Names {
			key := a.declare(name)
			if i < len(vs.Values) {
				a.attach(key, a.facts(vs.Values[i]))
			}
		}
	}
}

func (a *funcAnalyzer) assign(as *ast.AssignStmt) {
	// `flags := req.Flags` and `args := req.Args` make a local name mean the
	// thing the rest of the function indexes.
	for i, lhs := range as.Lhs {
		id, ok := lhs.(*ast.Ident)
		if !ok || i >= len(as.Rhs) {
			continue
		}
		if a.isFlagsExpr(as.Rhs[i]) {
			a.flagsIdents[id.Name] = true
		}
		if a.isArgsExpr(as.Rhs[i]) {
			a.argsIdents[id.Name] = true
		}
	}

	groups := make([]factGroup, len(as.Rhs))
	for i := range as.Rhs {
		groups[i] = a.facts(as.Rhs[i])
		if isConstruction(as.Rhs[i]) {
			// A struct built out of values is not one of those values. The
			// connection packages assemble everything into one object -- a
			// connection holding both an API key read from the environment and
			// a base URL read from an option -- and propagating through that
			// object would pair the two and report a credential route that does
			// not exist. The expression is still walked, so what it reads is
			// still recorded; only the flow into the name on the left stops.
			groups[i] = factGroup{}
		}
	}

	for i, lhs := range as.Lhs {
		id, ok := lhs.(*ast.Ident)
		if !ok {
			a.expr(lhs)
			continue
		}
		if id.Name == "_" {
			continue
		}
		var key string
		if as.Tok == token.DEFINE {
			key = a.declare(id)
		} else {
			key = a.resolve(id.Name)
			if _, known := a.vars[key]; !known {
				a.vars[key] = &varInfo{name: id.Name}
				a.order = append(a.order, key)
			}
		}
		switch {
		case len(groups) == len(as.Lhs):
			a.attach(key, groups[i])
		case len(groups) == 1:
			// A multi-value call or a comma-ok: every name on the left is fed
			// by the one expression on the right.
			a.attach(key, groups[0])
		}
	}
}

func (a *funcAnalyzer) attach(key string, g factGroup) {
	if len(g.flags) == 0 && len(g.envs) == 0 && len(g.refs) == 0 {
		return
	}
	v, ok := a.vars[key]
	if !ok {
		v = &varInfo{name: key}
		a.vars[key] = v
		a.order = append(a.order, key)
	}
	v.groups = append(v.groups, g)
}

// --- expressions --------------------------------------------------------

func (a *funcAnalyzer) expr(e ast.Expr) {
	if e == nil {
		return
	}
	a.facts(e)
}

// facts reads one expression for the four things that matter: flag names taken
// out of the flag map, environment variables read, other variables referenced,
// and positional comparisons.
func (a *funcAnalyzer) facts(e ast.Expr) factGroup {
	var g factGroup
	var visit func(n ast.Node) bool
	visit = func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncLit:
			// A closure is code that runs in this function's scope, so its
			// reads are this function's reads. hcp declares its whole flag
			// accessor as one.
			a.litDepth++
			a.push()
			a.declareParams(x.Type.Params)
			for _, st := range x.Body.List {
				a.stmt(st)
			}
			a.pop()
			a.litDepth--
			return false
		case *ast.IndexExpr:
			if !a.isFlagsExpr(x.X) {
				return true
			}
			name, ok := a.sy.str(x.Index)
			if !ok {
				if !a.localBinding(x.Index) {
					a.gap(GapUnresolvedConstant,
						fmt.Sprintf("%s indexes the flag map with something that did not resolve to a string", a.fn), x.Index)
				}
				return false
			}
			g.flags = appendUnique(g.flags, name)
			return false
		case *ast.CallExpr:
			if arg, ok := getenvArg(x); ok {
				name, ok := a.sy.str(arg)
				if !ok {
					if !a.localBinding(arg) {
						a.gap(GapDynamicGetenv,
							fmt.Sprintf("%s calls os.Getenv with a value the syntax does not name", a.fn), arg)
					}
					return false
				}
				g.envs = appendUnique(g.envs, name)
				a.envsSeen = appendUnique(a.envsSeen, name)
				return false
			}
			if id, ok := x.Fun.(*ast.Ident); ok {
				if sig, known := a.helpers[id.Name]; known {
					if fe, flag, ok := a.helperCall(sig, x); ok {
						if flag != "" {
							g.flags = appendUnique(g.flags, flag)
						}
						for _, v := range fe.Vars {
							g.envs = appendUnique(g.envs, v)
							a.envsSeen = appendUnique(a.envsSeen, v)
						}
						g.selected += len(fe.Vars)
						if fe.Flag != "" && len(fe.Vars) > 0 {
							a.direct = append(a.direct, fe)
						}
						return false
					}
				}
				if s, known := a.summaries[id.Name]; known {
					addAll(&g.flags, s.flags)
					for _, v := range s.envs {
						g.envs = appendUnique(g.envs, v)
						a.envsSeen = appendUnique(a.envsSeen, v)
					}
					g.selected += len(s.envs)
					return false
				}
			}
			// Any other call. A call with one value going in and one coming
			// out carries that value -- string(x.Value),
			// strings.TrimSpace(os.Getenv("X")), strings.TrimLeft(v, `\`) --
			// and its facts belong to whatever the result is assigned to. A
			// call with two values going in combines them into something new,
			// and portainer is where that matters: connect() calls
			// newClient(address, accessToken), and letting both flow into the
			// client made --address look as though it travelled in
			// PORTAINER_ACCESS_TOKEN. The arguments are still walked, so what
			// they read is still recorded; only the flow into the result stops.
			var carried []factGroup
			for _, arg := range x.Args {
				sub := a.facts(arg)
				if len(sub.flags) > 0 || len(sub.envs) > 0 || len(sub.refs) > 0 {
					carried = append(carried, sub)
				}
			}
			if len(carried) == 1 {
				addAll(&g.flags, carried[0].flags)
				addAll(&g.envs, carried[0].envs)
				addAll(&g.refs, carried[0].refs)
				g.selected += carried[0].selected
			}
			return false
		case *ast.BinaryExpr:
			if x.Op == token.EQL || x.Op == token.NEQ {
				a.comparePositional(x)
			}
			return true
		case *ast.SelectorExpr:
			// `x.Value` refers to x; the field name is not a variable.
			ast.Inspect(x.X, visit)
			return false
		case *ast.Ident:
			if x.Name == "_" || x.Name == "nil" {
				return false
			}
			g.refs = appendUnique(g.refs, a.resolve(x.Name))
			return false
		}
		return true
	}
	ast.Inspect(e, visit)
	return g
}

// helperCall reads a recognised accessor's arguments.
//
// It reports the flag name separately from the association because the two are
// separately useful: an accessor that only names a flag -- artifactory's
// flagValue -- still tells the caller which flag the returned value belongs to,
// and the caller's own os.Getenv completes the pair.
func (a *funcAnalyzer) helperCall(sig helperSig, call *ast.CallExpr) (fe FlagEnv, flag string, ok bool) {
	if sig.flagArg >= 0 && sig.flagArg < len(call.Args) {
		v, resolved := a.sy.str(call.Args[sig.flagArg])
		if !resolved {
			if !a.localBinding(call.Args[sig.flagArg]) {
				a.gap(GapUnresolvedConstant,
					fmt.Sprintf("%s passes a flag name to an accessor that did not resolve to a string", a.fn),
					call.Args[sig.flagArg])
			}
		} else {
			flag = v
		}
	}

	var vars []string
	readEnvArg := func(e ast.Expr) {
		if v, resolved := a.sy.str(e); resolved {
			vars = appendUnique(vars, v)
			return
		}
		if !a.localBinding(e) {
			a.gap(GapDynamicGetenv,
				fmt.Sprintf("%s passes an environment variable name the syntax does not name", a.fn), e)
		}
	}
	if sig.envVariadic && len(sig.envArgs) > 0 {
		for i := sig.envArgs[0]; i < len(call.Args); i++ {
			readEnvArg(call.Args[i])
		}
	} else {
		for _, idx := range sig.envArgs {
			if idx < len(call.Args) {
				readEnvArg(call.Args[idx])
			}
		}
	}

	if flag == "" && len(vars) == 0 {
		return FlagEnv{}, "", false
	}
	return FlagEnv{Flag: flag, Vars: vars, Func: a.fn, Via: a.key.via}, flag, true
}

// isFlagsExpr reports whether an expression is the CLI flag map.
func (a *funcAnalyzer) isFlagsExpr(e ast.Expr) bool {
	if id, ok := e.(*ast.Ident); ok {
		return a.flagsIdents[id.Name]
	}
	return readsField(e, a.key.field)
}

// isArgsExpr reports whether an expression is the positional argument slice.
func (a *funcAnalyzer) isArgsExpr(e ast.Expr) bool {
	if id, ok := e.(*ast.Ident); ok {
		return a.argsIdents[id.Name]
	}
	return readsField(e, "Args")
}

// pureStringFuncs are the wrappers a provider puts around a positional argument
// before comparing it. Seeing through them is what turns `switch
// strings.ToLower(req.Args[0])` into a vocabulary rather than a gap.
var pureStringFuncs = map[string]bool{
	"ToLower": true, "ToUpper": true, "TrimSpace": true, "Trim": true, "Title": true,
}

// argsIndex reports which positional argument an expression reads.
func (a *funcAnalyzer) argsIndex(e ast.Expr) (int, bool) {
	switch x := e.(type) {
	case *ast.ParenExpr:
		return a.argsIndex(x.X)
	case *ast.CallExpr:
		if sel, ok := x.Fun.(*ast.SelectorExpr); ok && pureStringFuncs[sel.Sel.Name] && len(x.Args) > 0 {
			return a.argsIndex(x.Args[0])
		}
	case *ast.IndexExpr:
		if !a.isArgsExpr(x.X) {
			return 0, false
		}
		n, ok := uintValue(x.Index)
		if !ok {
			return 0, false
		}
		return int(n), true
	}
	return 0, false
}

func (a *funcAnalyzer) comparePositional(be *ast.BinaryExpr) {
	for _, pair := range [][2]ast.Expr{{be.X, be.Y}, {be.Y, be.X}} {
		idx, ok := a.argsIndex(pair[0])
		if !ok {
			continue
		}
		v, ok := a.sy.str(pair[1])
		if !ok {
			a.gap(GapComputedPositional,
				fmt.Sprintf("%s compares positional argument %d against something other than a literal", a.fn, idx), pair[1])
			return
		}
		if v != "" {
			a.recordPositional(idx, v)
		}
		return
	}
}

func (a *funcAnalyzer) recordPositional(idx int, value string) {
	if _, seen := a.positional[idx]; !seen {
		a.posOrder = append(a.posOrder, idx)
	}
	a.positional[idx] = appendUnique(a.positional[idx], value)
}

// --- resolution ---------------------------------------------------------

// fixpoint propagates flag names and variable names along variable references
// until nothing more moves. The chains are short -- two or three hops -- so the
// naive repetition is cheaper than building a graph.
func (a *funcAnalyzer) fixpoint() {
	for range len(a.order) + 1 {
		changed := false
		for _, key := range a.order {
			v := a.vars[key]
			for _, g := range v.groups {
				changed = addAll(&v.flags, g.flags) || changed
				changed = addAll(&v.envs, g.envs) || changed
				for _, ref := range g.refs {
					src, ok := a.vars[ref]
					if !ok || src == v {
						continue
					}
					changed = addAll(&v.flags, src.flags) || changed
					changed = addAll(&v.envs, src.envs) || changed
				}
			}
		}
		if !changed {
			return
		}
	}
}

func contains(list []string, v string) bool {
	for _, existing := range list {
		if existing == v {
			return true
		}
	}
	return false
}

// isConstruction reports whether an expression assembles a new value out of
// other values, rather than reading or choosing one.
func isConstruction(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.CompositeLit:
		return true
	case *ast.UnaryExpr:
		return x.Op == token.AND && isConstruction(x.X)
	case *ast.CallExpr:
		id, ok := x.Fun.(*ast.Ident)
		return ok && id.Name == "append"
	}
	return false
}

// addAll unions src into dst, reporting whether anything moved.
func addAll(dst *[]string, src []string) bool {
	changed := false
	for _, v := range src {
		before := len(*dst)
		*dst = appendUnique(*dst, v)
		changed = changed || len(*dst) != before
	}
	return changed
}

// results are the flag-to-environment associations this function established.
//
// It is computed once: the walk raises gaps while resolving, and a second call
// would raise them again.
func (a *funcAnalyzer) results() []FlagEnv {
	if a.resolved {
		return a.cached
	}
	a.resolved = true
	a.cached = a.computeResults()
	return a.cached
}

func (a *funcAnalyzer) computeResults() []FlagEnv {
	out := append([]FlagEnv(nil), a.direct...)

	for _, key := range a.order {
		v := a.vars[key]
		if len(v.flags) == 0 || len(v.envs) == 0 {
			continue
		}
		if len(v.flags) > 1 {
			// Two flag names in one variable. Either could be the one the
			// variable travels for, and picking is exactly the failure mode
			// this tool exists to avoid.
			//
			// Unless nothing was lost: a value assembled from several already
			// paired flags -- activedirectory builds one credential out of
			// --user and --password after each has met its own variable -- is
			// an aggregate, not an unresolved binding, and reporting it would
			// bury the three real ones under the structures that carry them.
			if !a.allPairedElsewhere(v, out) {
				a.gaps = append(a.gaps, Gap{
					Kind: GapAmbiguousBinding,
					Detail: fmt.Sprintf("%s: %q holds flags %s and variables %s, so no pairing can be read",
						a.fn, v.name, strings.Join(v.flags, ", "), strings.Join(v.envs, ", ")),
				})
			}
			continue
		}
		out = append(out, FlagEnv{
			Flag:     v.flags[0],
			Vars:     v.envs,
			Func:     a.fn,
			Via:      a.key.via,
			Composed: a.composed(v),
		})
	}

	for _, chain := range a.branchChains {
		var flags, envs []string
		bothInOne := false
		for _, key := range chain.keys {
			v, ok := a.vars[key]
			if !ok {
				continue
			}
			if len(v.flags) > 0 && len(v.envs) > 0 {
				bothInOne = true
			}
			addAll(&flags, v.flags)
			addAll(&envs, v.envs)
		}
		if bothInOne || len(flags) != 1 || len(envs) == 0 {
			continue
		}
		for _, e := range envs {
			a.explained = appendUnique(a.explained, e)
		}
		a.gaps = append(a.gaps, Gap{
			Kind: GapAlternativeBranches,
			Detail: fmt.Sprintf("%s: %q is declared by two arms of one if/else chain, one holding flag %q and the other %s; nothing but the shared name says they are the same credential",
				a.fn, chain.name, flags[0], strings.Join(envs, ", ")),
		})
	}

	// Merge, keeping first-seen variable order.
	merged := map[string]*FlagEnv{}
	var order []string
	for _, fe := range out {
		existing, ok := merged[fe.Flag]
		if !ok {
			copied := fe
			merged[fe.Flag] = &copied
			order = append(order, fe.Flag)
			continue
		}
		for _, v := range fe.Vars {
			existing.Vars = appendUnique(existing.Vars, v)
		}
		existing.Composed = existing.Composed || fe.Composed
	}
	final := make([]FlagEnv, 0, len(order))
	for _, flag := range order {
		final = append(final, *merged[flag])
	}
	return final
}

// allPairedElsewhere reports whether every flag in an aggregate value already
// has its own association from this function.
func (a *funcAnalyzer) allPairedElsewhere(v *varInfo, found []FlagEnv) bool {
	for _, flag := range v.flags {
		paired := false
		for _, fe := range found {
			if fe.Flag == flag {
				paired = true
				break
			}
		}
		if !paired {
			return false
		}
	}
	return true
}

// composed reports whether a single assignment gathered more than one
// environment variable into this value, as opposed to the value being chosen
// from a list of fallbacks.
//
// The difference matters to anyone acting on the result. okta builds its
// organization from OKTA_ORG_NAME and OKTA_BASE_URL joined by a ".", so setting
// only the first produces "acme." and a connection to nothing; alicloud's
// region falls back from ALIBABA_CLOUD_REGION to ALIBABA_CLOUD_REGION_ID, and
// setting only the first is exactly right.
func (a *funcAnalyzer) composed(v *varInfo) bool {
	for _, g := range v.groups {
		count := len(g.envs) - g.selected
		for _, ref := range g.refs {
			if src, ok := a.vars[ref]; ok && src != v {
				count += len(src.envs)
			}
		}
		if count > 1 {
			return true
		}
	}
	return false
}

// boundLater reports whether an environment variable this function read is
// already accounted for -- either attached to a flag, or named by a more
// specific gap. It exists so the broad unbound-env report says only what
// nothing else has said.
func (a *funcAnalyzer) boundLater(name string) bool {
	for _, fe := range a.results() {
		for _, v := range fe.Vars {
			if v == name {
				return true
			}
		}
	}
	for _, v := range a.explained {
		if v == name {
			return true
		}
	}
	return false
}
