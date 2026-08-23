// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generate

import (
	"strings"

	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/mqlc"
	"go.mondoo.com/mql/types"
)

// propsHandler tells the compiler what `props.<name>` means. Without one, every
// prop reference is an unknown symbol and the query is rejected — which is what
// made the generator reject correct answers: the prompt asks the agent to
// reference props, and the compile gate then refused every query that did. All
// 49 prop-using checks in content/ fail to compile with a nil handler and all 49
// compile with this one.
//
// It mirrors policy.QueryPropsResolver (bundle.go), the handler the bundle
// compiler and `cnspec policy lint` use: a prop resolves to a bare primitive
// carrying only its *type*, because the compiler needs the type to pick an
// operator handler and nothing else.
type propsHandler struct {
	byName map[string]*llx.Primitive
}

// newPropsHandler resolves each prop to a type, in the same order of preference
// the bundle compiler uses:
//
//  1. an explicitly declared type,
//  2. the type of the prop's own MQL, compiled against the schema,
//  3. types.Any as a last resort — the prop name then resolves, though a typed
//     comparison against it will still fail ("cannot find operator handler"),
//     which is the honest outcome when the bundle never says what the prop is.
//
// It returns nil when there are no props, so callers pass a nil PropsHandler and
// the compiler keeps its "no props are available" behavior.
func newPropsHandler(props []Prop, conf mqlc.CompilerConfig) mqlc.PropsHandler {
	if len(props) == 0 {
		return nil
	}
	byName := make(map[string]*llx.Primitive, len(props))
	for _, p := range props {
		name := strings.TrimSpace(p.Name)
		if name == "" {
			continue // a prop with no name has no props.<name> to resolve
		}
		byName[name] = &llx.Primitive{Type: string(propType(p, conf))}
	}
	if len(byName) == 0 {
		return nil
	}
	return &propsHandler{byName: byName}
}

func (h *propsHandler) Get(name string) *llx.Primitive { return h.byName[name] }

func (h *propsHandler) Available() map[string]*llx.Primitive { return h.byName }

func (h *propsHandler) All() map[string]*llx.Primitive { return h.byName }

// propType resolves one prop's type. See newPropsHandler for the order.
func propType(p Prop, conf mqlc.CompilerConfig) types.Type {
	if t := parsePropType(p.Type); t != types.NoType {
		return t
	}
	if t := compiledType(p.Mql, conf); t != types.NoType {
		return t
	}
	return types.Any
}

// compiledType compiles a snippet and reports the type of its result. This is
// how a prop declared only as MQL (the shape used throughout content/) gets a
// type, and it is what internal/lsp's YAMLPropsHandler does for the same reason.
func compiledType(mql string, conf mqlc.CompilerConfig) types.Type {
	if strings.TrimSpace(mql) == "" {
		return types.NoType
	}
	bundle, err := mqlc.Compile(mql, nil, conf)
	if err != nil || bundle == nil || bundle.CodeV2 == nil {
		return types.NoType
	}
	entrypoints := bundle.CodeV2.Entrypoints()
	if len(entrypoints) == 0 {
		return types.NoType
	}
	chunk := bundle.CodeV2.Chunk(entrypoints[len(entrypoints)-1])
	if chunk == nil {
		return types.NoType
	}
	resolved := types.Type(chunk.DereferencedTypeV2(bundle.CodeV2))
	if resolved == types.Unset {
		// a snippet that compiles to nothing usable is no better than no type
		return types.NoType
	}
	return resolved
}

// propTypeAliases are the human-readable spellings a hand-written bundle may use
// in a prop's `type:` field. A compiled bundle instead stores the llx type code
// (a string of control bytes), which parsePropType passes through untouched.
var propTypeAliases = map[string]types.Type{
	"string":  types.String,
	"int":     types.Int,
	"integer": types.Int,
	"float":   types.Float,
	"double":  types.Float,
	"number":  types.Float,
	"bool":    types.Bool,
	"boolean": types.Bool,
	"time":    types.Time,
	"dict":    types.Dict,
	"regex":   types.Regex,
	"any":     types.Any,
}

// parsePropType turns a prop's declared type into an llx type. It accepts both
// the stored llx code and the human spellings above, including `[]x` and
// `map[string]x`, and returns NoType for anything it does not recognize so the
// caller falls through to the next source rather than compiling against a
// fabricated type. NoType is the "we do not know" answer rather than
// types.Unset, because Unset is itself a valid llx type (`\x00`): handing it to
// the compiler yields "cannot find field 'x' in unset" instead of falling back.
func parsePropType(declared string) types.Type {
	declared = strings.TrimSpace(declared)
	if declared == "" {
		return types.NoType
	}

	// A stored llx type is a string of control bytes, never printable text.
	if !isPrintableASCII(declared) {
		return types.Type(declared)
	}

	lower := strings.ToLower(declared)
	if t, ok := propTypeAliases[lower]; ok {
		return t
	}
	if rest, ok := strings.CutPrefix(lower, "[]"); ok {
		if child := parsePropType(rest); child != types.NoType {
			return types.Array(child)
		}
		return types.NoType
	}
	if rest, ok := strings.CutPrefix(lower, "map[string]"); ok {
		if child := parsePropType(rest); child != types.NoType {
			return types.Map(types.String, child)
		}
		return types.NoType
	}
	return types.NoType
}

func isPrintableASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7e {
			return false
		}
	}
	return true
}
