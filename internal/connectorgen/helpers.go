// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connectorgen

import (
	"go/ast"
)

// Almost no provider writes the flag-then-environment fallback out longhand.
// It writes a small accessor and calls it once per credential, and there are
// three shapes of accessor across the tree:
//
//	clientSecret := flagOrEnv(flags, "client-secret", "JAMF_CLIENT_SECRET")
//	if v := stringFlag(flags, "region", "ALIBABA_CLOUD_REGION", "ALIBABA_CLOUD_REGION_ID"); v != ""
//	token := flagValue(flags, "token"); if token == "" { token = os.Getenv("ARTIFACTORY_TOKEN") }
//
// The first two carry the whole fact in the call's arguments. The third carries
// only half -- the flag -- and leaves the variable to the caller, so a walk that
// only understood the first two would see artifactory read ARTIFACTORY_TOKEN
// into a value it could not name.
//
// So an accessor is recognised by what its body does with its own parameters,
// not by its name and not by whether it also reads the environment: a function
// that indexes the flag map by one of its parameters tells the call site which
// flag that argument names, and that is worth knowing on its own. Matching on
// the name would have found two of them; requiring an os.Getenv would have
// found six.
//
// Closures count. hcp declares its accessor inside ParseCLI and closes over the
// flag map rather than taking it as a parameter, so the flag map has to be
// recognised by name as well as by type.

// keySource says where an accessor's keys come from: the CLI flag map that
// ParseCLI is handed, or the connection options map that the connection package
// reads back at connect time.
//
// The second exists because roughly a fifth of the tree reads its credentials
// there rather than in ParseCLI. zoom's accessor is
//
//	getOptionValueFrom(options map[string]string, envVar string, option string)
//
// which is the same shape one package over, with the option name standing in
// for the flag name. Those two names are not the same thing in general, so an
// association found this way is kept only when the option name is also a flag
// the connector declares -- see assemble.
type keySource struct {
	// field is the request or config field the map is read from: "Flags" or
	// "Options".
	field string
	// typeOK recognises the map as a parameter type.
	typeOK func(ast.Expr) bool
	// via labels associations found this way in the artifact.
	via string
	// reportUnbound says whether an environment variable that meets no key is
	// worth a gap of its own. In ParseCLI it is; in the connection package the
	// caller reports the whole set at once instead, because half of them are
	// read for reasons that have nothing to do with a flag.
	reportUnbound bool
}

// flagKeys reads the CLI flag map inside a provider package.
var flagKeys = keySource{field: "Flags", typeOK: isFlagMapType, via: "parse-cli", reportUnbound: true}

// optionKeys reads the connection options map inside a connection package.
var optionKeys = keySource{field: "Options", typeOK: isOptionMapType, via: "connection"}

// helperSig describes a recognised flag accessor.
type helperSig struct {
	// flagArg is the parameter position holding the flag name, or -1 when the
	// accessor names no flag.
	flagArg int
	// envArgs are the parameter positions holding environment variable names,
	// in the order the body consults them.
	envArgs []int
	// envVariadic marks an accessor whose last parameter is `envs ...string`,
	// so every argument from envArgs[0] onward is a variable name.
	envVariadic bool
}

// findHelpers recognises the flag accessors declared in a package, whether as
// functions or as closures assigned to a name.
func findHelpers(files []*ast.File, key keySource) map[string]helperSig {
	flagMaps := flagMapNames(files, key)
	out := map[string]helperSig{}
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				if x.Body == nil || x.Recv != nil {
					return true
				}
				if sig, ok := helperSignature(x.Type.Params, x.Body, flagMaps, key); ok {
					out[x.Name.Name] = sig
				}
			case *ast.AssignStmt:
				// `flagOrEnv := func(name, env string) string { ... }`
				for i, rhs := range x.Rhs {
					lit, ok := rhs.(*ast.FuncLit)
					if !ok || i >= len(x.Lhs) {
						continue
					}
					id, ok := x.Lhs[i].(*ast.Ident)
					if !ok {
						continue
					}
					if sig, ok := helperSignature(lit.Type.Params, lit.Body, flagMaps, key); ok {
						out[id.Name] = sig
					}
				}
			}
			return true
		})
	}
	return out
}

// flagMapNames collects the identifiers a package uses for the CLI flag map, so
// a closure that captures one can be recognised. It is deliberately generous:
// a name wrongly included here can only cause an accessor to be recognised that
// takes a flag name it does not have, and the flag would then fail to match any
// declared flag and be reported rather than believed.
func flagMapNames(files []*ast.File, key keySource) map[string]bool {
	out := map[string]bool{}
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.AssignStmt:
				for i, rhs := range x.Rhs {
					if i >= len(x.Lhs) {
						break
					}
					if !readsField(rhs, key.field) {
						continue
					}
					if id, ok := x.Lhs[i].(*ast.Ident); ok {
						out[id.Name] = true
					}
				}
			case *ast.FuncDecl:
				for _, p := range flatParams(x.Type.Params) {
					if p.name != "" && key.typeOK(p.typ) {
						out[p.name] = true
					}
				}
			}
			return true
		})
	}
	return out
}

// helperSignature reports whether a function body reads a flag out of the CLI
// flag map using one of its own parameters as the key.
func helperSignature(params *ast.FieldList, body *ast.BlockStmt, flagMaps map[string]bool, key keySource) (helperSig, bool) {
	flat := flatParams(params)
	if len(flat) == 0 || body == nil {
		return helperSig{}, false
	}

	byName := map[string]int{}
	variadic := map[string]int{}
	local := map[string]bool{}
	for i, p := range flat {
		if p.name == "" {
			continue
		}
		byName[p.name] = i
		if p.variadic {
			variadic[p.name] = i
		}
		if key.typeOK(p.typ) {
			local[p.name] = true
		}
	}
	isFlagMap := func(e ast.Expr) bool {
		if id, ok := e.(*ast.Ident); ok {
			return local[id.Name] || flagMaps[id.Name]
		}
		return readsField(e, key.field)
	}

	// Range variables bound to a parameter, so `for _, e := range envs` makes
	// os.Getenv(e) a read of the envs parameter.
	rangeOf := map[string]int{}
	ast.Inspect(body, func(n ast.Node) bool {
		rs, ok := n.(*ast.RangeStmt)
		if !ok {
			return true
		}
		src, ok := rs.X.(*ast.Ident)
		if !ok {
			return true
		}
		idx, ok := byName[src.Name]
		if !ok {
			return true
		}
		if v, ok := rs.Value.(*ast.Ident); ok {
			rangeOf[v.Name] = idx
		}
		return true
	})

	sig := helperSig{flagArg: -1}
	ast.Inspect(body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncLit:
			// A nested closure has its own parameters; attributing its reads to
			// this signature's argument positions would be wrong.
			return false
		case *ast.IndexExpr:
			if !isFlagMap(x.X) {
				return true
			}
			if key, ok := x.Index.(*ast.Ident); ok {
				if idx, ok := byName[key.Name]; ok && sig.flagArg == -1 {
					sig.flagArg = idx
				}
			}
		case *ast.CallExpr:
			arg, ok := getenvArg(x)
			if !ok {
				return true
			}
			// `os.Getenv(envs[i])` is a read of the envs parameter just as
			// `os.Getenv(e)` after a range over it is.
			if ix, isIndex := arg.(*ast.IndexExpr); isIndex {
				arg = ix.X
			}
			id, ok := arg.(*ast.Ident)
			if !ok {
				return true
			}
			if idx, ok := byName[id.Name]; ok {
				sig.envArgs = appendUnique(sig.envArgs, idx)
				if _, isVariadic := variadic[id.Name]; isVariadic {
					sig.envVariadic = true
				}
				return true
			}
			if idx, ok := rangeOf[id.Name]; ok {
				sig.envArgs = appendUnique(sig.envArgs, idx)
				for _, vidx := range variadic {
					if vidx == idx {
						sig.envVariadic = true
					}
				}
			}
		}
		return true
	})

	if sig.flagArg == -1 && len(sig.envArgs) == 0 {
		return helperSig{}, false
	}
	return sig, true
}

// funcSummary is what calling a package-local function contributes when the
// values it returns are named in its own body rather than passed in.
//
// mistral is the case that needs it: `envToken()` takes no arguments and
// returns MISTRAL_API_KEY or MISTRAL_KEY, and ParseCLI writes `if token == ""
// { token = envToken() }`. Without following the call, the variable holding
// --token never meets a variable name and mistral reports as having no
// credential route at all.
type funcSummary struct {
	flags []string
	envs  []string
}

// stringReturning reports whether every result of a function is a string. The
// summary is applied only to those: a function returning a connection or an
// asset also reads the environment for its own reasons, and letting those reads
// flow into a caller's variable is how a wrong credential route gets invented.
func stringReturning(results *ast.FieldList) bool {
	if results == nil || len(results.List) == 0 {
		return false
	}
	for _, f := range results.List {
		id, ok := f.Type.(*ast.Ident)
		if !ok || id.Name != "string" {
			return false
		}
	}
	return true
}

// param is one flattened function parameter.
type param struct {
	name     string
	typ      ast.Expr
	variadic bool
}

// flatParams expands `flag, env string` into one entry per name, which is what
// an argument position means at a call site.
func flatParams(fl *ast.FieldList) []param {
	if fl == nil {
		return nil
	}
	var out []param
	for _, f := range fl.List {
		typ := f.Type
		isVariadic := false
		if ell, ok := typ.(*ast.Ellipsis); ok {
			typ = ell.Elt
			isVariadic = true
		}
		if len(f.Names) == 0 {
			out = append(out, param{typ: typ, variadic: isVariadic})
			continue
		}
		for _, n := range f.Names {
			out = append(out, param{name: n.Name, typ: typ, variadic: isVariadic})
		}
	}
	return out
}

// readsField reports whether an expression reads a named field of a request:
// either `req.Flags` or the generated getter `req.GetFlags()`. Both spellings
// are in the tree, and activedirectory uses only the second.
func readsField(e ast.Expr, field string) bool {
	switch x := e.(type) {
	case *ast.SelectorExpr:
		return x.Sel.Name == field
	case *ast.CallExpr:
		sel, ok := x.Fun.(*ast.SelectorExpr)
		return ok && len(x.Args) == 0 && sel.Sel.Name == "Get"+field
	}
	return false
}

// isFlagMapType reports whether a parameter type is the CLI flag map,
// map[string]*llx.Primitive.
//
// The pointer element is what makes this a test rather than a guess: the os
// provider passes map[string]struct{} around, and a rule that accepted any map
// read `detectors[name]` as a flag lookup.
func isFlagMapType(t ast.Expr) bool {
	m, ok := t.(*ast.MapType)
	if !ok {
		return false
	}
	if key, ok := m.Key.(*ast.Ident); !ok || key.Name != "string" {
		return false
	}
	_, isPtr := m.Value.(*ast.StarExpr)
	return isPtr
}

// isOptionMapType reports whether a parameter type is the connection options
// map, map[string]string.
func isOptionMapType(t ast.Expr) bool {
	m, ok := t.(*ast.MapType)
	if !ok {
		return false
	}
	key, ok := m.Key.(*ast.Ident)
	if !ok || key.Name != "string" {
		return false
	}
	val, ok := m.Value.(*ast.Ident)
	return ok && val.Name == "string"
}

// getenvArg returns the argument of an os.Getenv call.
func getenvArg(call *ast.CallExpr) (ast.Expr, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Getenv" {
		return nil, false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok || pkg.Name != "os" {
		return nil, false
	}
	if len(call.Args) != 1 {
		return nil, false
	}
	return call.Args[0], true
}

func appendUnique[T comparable](list []T, v T) []T {
	for _, existing := range list {
		if existing == v {
			return list
		}
	}
	return append(list, v)
}
