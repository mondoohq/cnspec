// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connectorgen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

// A provider's config literal is full of identifiers rather than strings:
// `Discovery: []string{connection.DiscoveryRepos, ...}`, `flags[connection.
// OPTION_APP_ID]`. Resolving them is the difference between extracting github's
// ten discovery targets and extracting ten blanks.
//
// With no type checker the resolution is by name, over every package in the
// provider's own tree. That is sound as long as a name means one thing there,
// and where it does not -- the same identifier declared twice with different
// values in two sibling packages -- the table refuses to answer rather than
// picking one, and the caller records an unresolved-constant gap.

// symbols is a by-name table of the string and []string constants declared
// anywhere in one provider's source tree.
type symbols struct {
	// strs maps both "Name" and "pkg.Name" to a string value.
	strs map[string]string
	// lists maps the same keys to a []string value.
	lists map[string][]string
	// ambiguous holds keys that resolved to more than one distinct value, and
	// which the table therefore declines to answer for.
	ambiguous map[string]bool
	// files are the parsed files, keyed by directory, so a caller can walk the
	// same trees without re-reading them.
	files map[string][]*ast.File
	fset  *token.FileSet
}

func newSymbols(fset *token.FileSet) *symbols {
	return &symbols{
		strs:      map[string]string{},
		lists:     map[string][]string{},
		ambiguous: map[string]bool{},
		files:     map[string][]*ast.File{},
		fset:      fset,
	}
}

// scanTree parses every non-test Go file under root and records the string
// constants it finds. Unparseable files are skipped rather than fatal: a
// provider tree pinned to an older SDK may hold a file this toolchain cannot
// read, and losing one file's constants is better than losing the provider.
func (s *symbols) scanTree(root string) error {
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Vendored trees and testdata hold other people's constants.
			switch d.Name() {
			case "vendor", "testdata", ".git":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, perr := parseOne(s.fset, path)
		if perr != nil {
			return nil
		}
		dir := filepath.Dir(path)
		s.files[dir] = append(s.files[dir], f)
		s.record(f)
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "cannot walk %s", root)
	}
	return nil
}

// filesUnder returns every parsed file in a directory and its subdirectories,
// in a stable order.
//
// The packages under connection/ are separate packages, so this mixes them.
// That is deliberate and safe for what it is used for: the walk over them looks
// for an option name paired with an environment variable, and both are string
// literals rather than cross-package references.
func (s *symbols) filesUnder(root string) []*ast.File {
	var dirs []string
	for dir := range s.files {
		if dir == root || strings.HasPrefix(dir, root+string(filepath.Separator)) {
			dirs = append(dirs, dir)
		}
	}
	sort.Strings(dirs)
	var out []*ast.File
	for _, dir := range dirs {
		out = append(out, s.files[dir]...)
	}
	return out
}

// parseOne parses a single file for its syntax only. Object resolution is
// skipped because nothing here needs it and it is the expensive half.
func parseOne(fset *token.FileSet, path string) (*ast.File, error) {
	return parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
}

// record adds one file's package-level string and []string declarations.
func (s *symbols) record(f *ast.File) {
	pkg := f.Name.Name
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || (gd.Tok != token.CONST && gd.Tok != token.VAR) {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range vs.Names {
				if i >= len(vs.Values) {
					continue
				}
				if v, ok := literalString(vs.Values[i]); ok {
					s.put(pkg, name.Name, v)
					continue
				}
				if v, ok := literalStringList(vs.Values[i]); ok {
					s.putList(pkg, name.Name, v)
				}
			}
		}
	}
}

func (s *symbols) put(pkg, name, val string) {
	for _, key := range []string{name, pkg + "." + name} {
		if old, seen := s.strs[key]; seen && old != val {
			s.ambiguous[key] = true
			continue
		}
		s.strs[key] = val
	}
}

func (s *symbols) putList(pkg, name string, val []string) {
	for _, key := range []string{name, pkg + "." + name} {
		if old, seen := s.lists[key]; seen && !equalStrings(old, val) {
			s.ambiguous[key] = true
			continue
		}
		s.lists[key] = val
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// str resolves an expression to a string.
//
// It handles the three shapes a provider config actually uses -- a literal, an
// identifier or qualified identifier naming a constant, and a concatenation of
// those -- and refuses everything else. Refusing is the point: a name it cannot
// resolve becomes a gap, not an empty string that reads like a declared empty
// value.
func (s *symbols) str(e ast.Expr) (string, bool) {
	switch x := e.(type) {
	case *ast.BasicLit:
		return literalString(x)
	case *ast.Ident:
		return s.lookup(x.Name)
	case *ast.SelectorExpr:
		if pkg, ok := x.X.(*ast.Ident); ok {
			return s.lookup(pkg.Name + "." + x.Sel.Name)
		}
		return "", false
	case *ast.ParenExpr:
		return s.str(x.X)
	case *ast.CallExpr:
		// fmt.Sprintf, which is how seven providers write a Long help text that
		// names the environment variables it reads: the variable names are
		// constants passed as arguments. Resolving the call is what turns that
		// help text from an unresolved blank into the sentence a user reads.
		// string(pkg.Const), which is how a provider compares a positional
		// argument against a typed constant. atlassian spells one of its four
		// sub-commands that way, and without this its vocabulary comes out
		// three words long with no sign that a fourth was dropped.
		if id, ok := x.Fun.(*ast.Ident); ok && id.Name == "string" && len(x.Args) == 1 {
			return s.str(x.Args[0])
		}
		sel, ok := x.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Sprintf" {
			return "", false
		}
		if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "fmt" || len(x.Args) == 0 {
			return "", false
		}
		format, ok := s.str(x.Args[0])
		if !ok {
			return "", false
		}
		args := make([]any, 0, len(x.Args)-1)
		for _, a := range x.Args[1:] {
			v, ok := s.str(a)
			if !ok {
				return "", false
			}
			args = append(args, v)
		}
		return fmt.Sprintf(format, args...), true
	case *ast.BinaryExpr:
		if x.Op != token.ADD {
			return "", false
		}
		left, leftOK := s.str(x.X)
		right, rightOK := s.str(x.Y)
		if !leftOK || !rightOK {
			return "", false
		}
		return left + right, true
	}
	return "", false
}

func (s *symbols) lookup(key string) (string, bool) {
	if s.ambiguous[key] {
		return "", false
	}
	v, ok := s.strs[key]
	return v, ok
}

// strList resolves an expression to a list of strings, reporting the entries it
// could resolve and whether every entry resolved. A partly resolved list is
// still worth keeping -- nine of ten discovery targets is nine more than none
// -- so the caller records the shortfall as a gap and uses what it got.
func (s *symbols) strList(e ast.Expr) (out []string, complete bool) {
	complete = true
	switch x := e.(type) {
	case *ast.CompositeLit:
		for _, el := range x.Elts {
			if v, ok := s.str(el); ok {
				out = append(out, v)
				continue
			}
			if nested, sub := s.strList(el); len(nested) > 0 {
				out = append(out, nested...)
				complete = complete && sub
				continue
			}
			complete = false
		}
		return out, complete
	case *ast.Ident:
		if v, ok := s.lists[x.Name]; ok && !s.ambiguous[x.Name] {
			return append([]string(nil), v...), true
		}
	case *ast.SelectorExpr:
		if pkg, ok := x.X.(*ast.Ident); ok {
			key := pkg.Name + "." + x.Sel.Name
			if v, ok := s.lists[key]; ok && !s.ambiguous[key] {
				return append([]string(nil), v...), true
			}
		}
	case *ast.CallExpr:
		// append(base, extra...) shows up in a couple of discovery lists.
		if id, ok := x.Fun.(*ast.Ident); ok && id.Name == "append" {
			for _, arg := range x.Args {
				if v, ok := s.str(arg); ok {
					out = append(out, v)
					continue
				}
				nested, sub := s.strList(arg)
				out = append(out, nested...)
				complete = complete && sub
			}
			return out, complete
		}
	}
	return nil, false
}

// literalString unquotes a string literal.
func literalString(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	v, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return v, true
}

// literalStringList reads a []string{...} of literals.
func literalStringList(e ast.Expr) ([]string, bool) {
	cl, ok := e.(*ast.CompositeLit)
	if !ok {
		return nil, false
	}
	at, ok := cl.Type.(*ast.ArrayType)
	if !ok {
		return nil, false
	}
	if id, ok := at.Elt.(*ast.Ident); !ok || id.Name != "string" {
		return nil, false
	}
	out := make([]string, 0, len(cl.Elts))
	for _, el := range cl.Elts {
		v, ok := literalString(el)
		if !ok {
			return nil, false
		}
		out = append(out, v)
	}
	return out, true
}

// uintValue reads an unsigned integer literal.
func uintValue(e ast.Expr) (uint, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.INT {
		return 0, false
	}
	v, err := strconv.ParseUint(lit.Value, 0, 64)
	if err != nil {
		return 0, false
	}
	return uint(v), true
}

// boolValue reads a bool literal.
func boolValue(e ast.Expr) (bool, bool) {
	id, ok := e.(*ast.Ident)
	if !ok {
		return false, false
	}
	switch id.Name {
	case "true":
		return true, true
	case "false":
		return false, true
	}
	return false, false
}

// flagTypes and flagOptions map the SDK's enum identifiers to their values.
//
// They are written out rather than reflected over because the mapping is what
// makes the artifact comparable to runtime metadata, and because naming the
// constants here means a rename in the SDK breaks this build instead of
// silently producing a flag with type 0. The plugin package is the SDK, which
// cnspec already depends on; it is the providers that cannot be imported.
var flagTypes = map[string]plugin.FlagType{
	"FlagType_Bool":     plugin.FlagType_Bool,
	"FlagType_Int":      plugin.FlagType_Int,
	"FlagType_String":   plugin.FlagType_String,
	"FlagType_List":     plugin.FlagType_List,
	"FlagType_KeyValue": plugin.FlagType_KeyValue,
}

var flagOptions = map[string]plugin.FlagOption{
	"FlagOption_Hidden":     plugin.FlagOption_Hidden,
	"FlagOption_Deprecated": plugin.FlagOption_Deprecated,
	"FlagOption_Required":   plugin.FlagOption_Required,
	"FlagOption_Password":   plugin.FlagOption_Password,
	"FlagOption_AskInput":   plugin.FlagOption_AskInput,
}

// enumValue resolves plugin.FlagType_String, or an OR of flag options, to its
// numeric value.
func enumValue[T ~byte](e ast.Expr, table map[string]T) (T, bool) {
	var zero T
	switch x := e.(type) {
	case *ast.SelectorExpr:
		v, ok := table[x.Sel.Name]
		return v, ok
	case *ast.Ident:
		v, ok := table[x.Name]
		return v, ok
	case *ast.ParenExpr:
		return enumValue(x.X, table)
	case *ast.BinaryExpr:
		if x.Op != token.OR {
			return zero, false
		}
		left, leftOK := enumValue(x.X, table)
		right, rightOK := enumValue(x.Y, table)
		if !leftOK || !rightOK {
			return zero, false
		}
		return left | right, true
	}
	return zero, false
}
