// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connectorgen

import (
	"fmt"
	"go/ast"
	"sort"
)

// Reading the plugin.Provider literal out of providers/<name>/config/config.go.
//
// Everything here is also obtainable from an installed provider's <name>.json,
// which is a verbatim json.Marshal of the same struct. It is extracted anyway
// so one artifact describes every connector rather than only the ones the
// person running the generator happened to have installed, and so the env and
// positional facts -- which have no runtime source -- sit beside the flags they
// belong to.

// declaredProvider is one parsed plugin.Provider literal.
type declaredProvider struct {
	// Name is the provider's own Name field, which is not always its directory
	// name: the provider in providers/claude declares itself "claude".
	Name       string
	Connectors []Connector
	Gaps       []Gap
}

// parseConfig reads the plugin.Provider literal from a package's files.
//
// It returns ok=false when no such literal is there at all, which is not a gap:
// most directories under providers/ hold no config, and a caller that reported
// each of them would bury the gaps that matter.
func parseConfig(sy *symbols, files []*ast.File, where func(ast.Node) string) (declaredProvider, bool) {
	var lit *ast.CompositeLit
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			if lit != nil {
				return false
			}
			cl, ok := n.(*ast.CompositeLit)
			if !ok || !isQualifiedType(cl.Type, "plugin", "Provider") {
				return true
			}
			lit = cl
			return false
		})
		if lit != nil {
			break
		}
	}
	if lit == nil {
		return declaredProvider{}, false
	}

	var out declaredProvider
	gap := func(kind, detail string, n ast.Node) {
		out.Gaps = append(out.Gaps, Gap{Kind: kind, Detail: detail, Where: where(n)})
	}

	for _, field := range fieldsOf(lit) {
		switch field.name {
		case "Name":
			v, ok := sy.str(field.value)
			if !ok {
				gap(GapUnresolvedConstant, "provider Name is not a resolvable string", field.value)
				continue
			}
			out.Name = v
		case "Connectors":
			cl, ok := field.value.(*ast.CompositeLit)
			if !ok {
				gap(GapConfigNotLiteral, "Connectors is not a composite literal, so no connector could be read", field.value)
				continue
			}
			for _, el := range cl.Elts {
				ecl, ok := unwrapCompositeLit(el)
				if !ok {
					gap(GapConfigNotLiteral, "a Connectors entry is not a composite literal", el)
					continue
				}
				out.Connectors = append(out.Connectors, parseConnector(sy, ecl, where, &out.Gaps))
			}
		}
	}
	// Every gap raised while reading the literal belongs to this provider.
	for i := range out.Gaps {
		out.Gaps[i].Provider = out.Name
	}
	return out, true
}

// parseConnector reads one plugin.Connector literal.
func parseConnector(sy *symbols, lit *ast.CompositeLit, where func(ast.Node) string, gaps *[]Gap) Connector {
	var c Connector
	gap := func(kind, detail string, n ast.Node) {
		*gaps = append(*gaps, Gap{Connector: c.Name, Kind: kind, Detail: detail, Where: where(n)})
	}
	// Name first, so every gap raised below can name the connector it is about.
	for _, field := range fieldsOf(lit) {
		if field.name != "Name" {
			continue
		}
		if v, ok := sy.str(field.value); ok {
			c.Name = v
		}
	}

	for _, field := range fieldsOf(lit) {
		switch field.name {
		case "Name":
			if c.Name == "" {
				gap(GapUnresolvedConstant, "connector Name is not a resolvable string", field.value)
			}
		case "Use":
			c.Use = mustStr(sy, field.value, "Use", gap)
		case "Short":
			c.Short = mustStr(sy, field.value, "Short", gap)
		case "Long":
			c.Long = mustStr(sy, field.value, "Long", gap)
		case "Maturity":
			c.Maturity = mustStr(sy, field.value, "Maturity", gap)
		case "MinArgs":
			if v, ok := uintValue(field.value); ok {
				c.MinArgs = v
			} else {
				gap(GapConfigNotLiteral, "MinArgs is not an integer literal", field.value)
			}
		case "MaxArgs":
			if v, ok := uintValue(field.value); ok {
				c.MaxArgs = v
			} else {
				gap(GapConfigNotLiteral, "MaxArgs is not an integer literal", field.value)
			}
		case "IsHidden":
			if v, ok := boolValue(field.value); ok {
				c.IsHidden = v
			}
		case "Aliases":
			v, complete := sy.strList(field.value)
			c.Aliases = v
			if !complete {
				gap(GapUnresolvedConstant, "some Aliases entries did not resolve to strings", field.value)
			}
		case "Discovery":
			v, complete := sy.strList(field.value)
			c.Discovery = v
			if !complete {
				gap(GapUnresolvedConstant, "some Discovery entries did not resolve to strings", field.value)
			}
		case "Flags":
			cl, ok := field.value.(*ast.CompositeLit)
			if !ok {
				gap(GapConfigNotLiteral, "Flags is not a composite literal, so no flag could be read", field.value)
				continue
			}
			for _, el := range cl.Elts {
				fcl, ok := unwrapCompositeLit(el)
				if !ok {
					gap(GapConfigNotLiteral, "a Flags entry is not a composite literal", el)
					continue
				}
				c.Flags = append(c.Flags, parseFlag(sy, fcl, where, gap))
			}
		}
	}
	// Sorted by name, matching the order the launcher's own snapshot records,
	// so a regeneration diff shows what changed rather than how the author
	// reordered the literal.
	sort.Slice(c.Flags, func(i, j int) bool { return c.Flags[i].Long < c.Flags[j].Long })
	return c
}

func parseFlag(sy *symbols, lit *ast.CompositeLit, where func(ast.Node) string, gap func(kind, detail string, n ast.Node)) Flag {
	var fl Flag
	for _, field := range fieldsOf(lit) {
		switch field.name {
		case "Long":
			fl.Long = mustStr(sy, field.value, "flag Long", gap)
		case "Short":
			fl.Short = mustStr(sy, field.value, "flag Short", gap)
		case "Default":
			fl.Default = mustStr(sy, field.value, "flag Default", gap)
		case "Desc":
			fl.Desc = mustStr(sy, field.value, "flag Desc", gap)
		case "ConfigEntry":
			fl.Config = mustStr(sy, field.value, "flag ConfigEntry", gap)
		case "Type":
			v, ok := enumValue(field.value, flagTypes)
			if !ok {
				gap(GapUnresolvedConstant, fmt.Sprintf("flag %q has a Type this walk cannot read", fl.Long), field.value)
				continue
			}
			fl.Type = int32(v)
		case "Option":
			v, ok := enumValue(field.value, flagOptions)
			if !ok {
				gap(GapUnresolvedConstant, fmt.Sprintf("flag %q has an Option this walk cannot read", fl.Long), field.value)
				continue
			}
			fl.Option = int32(v)
		}
	}
	_ = where
	return fl
}

// mustStr resolves a string field, recording a gap when it cannot. An empty
// value from an unresolved identifier and a declared empty value are different
// facts, and collapsing them is how a form ends up with a blank label nobody
// can explain.
func mustStr(sy *symbols, e ast.Expr, what string, gap func(kind, detail string, n ast.Node)) string {
	v, ok := sy.str(e)
	if !ok {
		gap(GapUnresolvedConstant, what+" is not a resolvable string", e)
		return ""
	}
	return v
}

// keyed is one Key: Value pair of a composite literal.
type keyed struct {
	name  string
	value ast.Expr
}

// fieldsOf returns the keyed fields of a struct literal, in source order.
func fieldsOf(lit *ast.CompositeLit) []keyed {
	out := make([]keyed, 0, len(lit.Elts))
	for _, el := range lit.Elts {
		kv, ok := el.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		out = append(out, keyed{name: key.Name, value: kv.Value})
	}
	return out
}

// unwrapCompositeLit reaches the literal behind an optional &.
func unwrapCompositeLit(e ast.Expr) (*ast.CompositeLit, bool) {
	if u, ok := e.(*ast.UnaryExpr); ok {
		e = u.X
	}
	cl, ok := e.(*ast.CompositeLit)
	return cl, ok
}

// isQualifiedType reports whether a literal's type is pkg.Name, allowing for a
// pointer or a slice element.
func isQualifiedType(e ast.Expr, pkg, name string) bool {
	switch x := e.(type) {
	case *ast.SelectorExpr:
		id, ok := x.X.(*ast.Ident)
		return ok && id.Name == pkg && x.Sel.Name == name
	case *ast.StarExpr:
		return isQualifiedType(x.X, pkg, name)
	}
	return false
}
