// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package yacit

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"google.golang.org/protobuf/reflect/protoreflect"
	"gopkg.in/yaml.v3"
)

const disclaimer = `// Code generated by yac-it. DO NOT EDIT.
//
// Configure yac-it for things you want to auto-generate and extend generated
// objects in a separate file please.

`

const prefix = `
import (
	"gopkg.in/yaml.v3"
	"encoding/json"
	"errors"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnspec/policy"
)

type FileContext struct {
	Line int
	Column int
}

`

func New(conf YacItConfig) *YacIt {
	res := &YacIt{
		types:           map[string]string{},
		io:              os.Stderr,
		customUnmarshal: map[string]struct{}{},
		pkg:             conf.Package,
		fieldOrder:      conf.FieldOrder,
	}

	for _, v := range conf.SkipUnmarshal {
		res.customUnmarshal[v] = struct{}{}
	}

	return res
}

type YacItConfig struct {
	SkipUnmarshal []string
	Package       string
	FieldOrder    map[string]int
}

type YacIt struct {
	types           map[string]string
	io              io.Writer
	customUnmarshal map[string]struct{}
	pkg             string
	fieldOrder      map[string]int
}

func (t *YacIt) Add(typ interface{}) {
	t.createStruct(reflect.TypeOf(typ).Elem())
}

func (t *YacIt) String() string {
	var res strings.Builder

	res.WriteString(disclaimer)

	if t.pkg != "" {
		res.WriteString("package " + t.pkg + "\n")
	}

	res.WriteString(prefix)

	keys := make([]string, len(t.types))
	i := 0
	for k := range t.types {
		keys[i] = k
		i++
	}
	sort.Strings(keys)

	for i := range keys {
		v := t.types[keys[i]]
		res.WriteString(v)
		res.WriteByte('\n')
	}

	return res.String()
}

func (t *YacIt) createStruct(typ reflect.Type) {
	name := typ.Name()

	if _, ok := t.types[name]; ok {
		return
	}

	var res strings.Builder
	res.WriteString("type " + name + " struct {\n")

	nuFields := make([]reflect.StructField, 0)
	for i := 0; i < typ.NumField(); i++ {
		cur := typ.Field(i)

		// TODO: limited to ascii right now, no utf8 runes
		if unicode.IsLower(rune(cur.Name[0])) {
			continue
		}

		nuFields = append(nuFields, cur)
		nuTag := addYamlTag(cur.Tag)
		nuFields[len(nuFields)-1].Tag = (nuTag)
	}

	// sort nuFields
	sort.SliceStable(nuFields, func(i, j int) bool {
		name1 := nuFields[i].Name
		name2 := nuFields[j].Name
		// check weights
		w1, ok1 := t.fieldOrder[name1]
		w2, ok2 := t.fieldOrder[name2]
		if ok1 && ok2 {
			return w1 > w2
		}
		// the entry with a weight is always greater
		if ok1 || ok2 {
			return true
		}

		return false
	})

	// render fields
	for i := range nuFields {
		field := nuFields[i]

		res.WriteByte('\t')
		res.WriteString(field.Name)
		res.WriteByte(' ')
		res.WriteString(printType(field.Type))
		res.WriteByte(' ')
		res.WriteByte('`')
		res.WriteString(string(field.Tag))
		res.WriteByte('`')
		res.WriteByte('\n')
	}

	res.WriteString("\tFileContext FileContext `json:\"-\" yaml:\"-\"`\n")
	res.WriteString("}\n")

	if _, isCustom := t.customUnmarshal[name]; !isCustom {
		res.WriteString(fmt.Sprintf(`
func (x *%s) UnmarshalYAML(node *yaml.Node) error {
	// prevent recursive calls into UnmarshalYAML with a placeholder type
	type tmp %s
	err := node.Decode((*tmp)(x))
	if err != nil {
		return err
	}

	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
	return nil
}
`, name, name))
	} else {
		res.WriteString(fmt.Sprintf(`
func (x *%s) addFileContext(node *yaml.Node) {
	x.FileContext.Column = node.Column
	x.FileContext.Line = node.Line
}
`, name))
	}

	t.types[name] = res.String()

	for i := range nuFields {
		typ := baseType(nuFields[i].Type)
		switch typ.Kind() {
		case reflect.Struct:
			t.io.Write([]byte("process type: " + typ.Name() + "\n"))
			t.createStruct(typ)
		default:
			// If the type is a protobuf enum, we allow it to be a string or int
			if shouldGenerateTypeForEnum(typ) {
				t.createProtoEnum(typ)
			}
		}
	}
}

func isProtoEnum(typ reflect.Type) bool {
	return typ.Implements(reflect.TypeOf((*ProtoEnum)(nil)).Elem())
}

var protoEnumTemplate = template.Must(template.New("protoEnum").Parse(`
func (s *{{.Name}}) UnmarshalYAML(node *yaml.Node) error {

	var decoded interface{}
	err := node.Decode(&decoded)
	if err != nil {
		return err
	}

	jsonData, err := json.Marshal(decoded)
	if err != nil {
		return err
	}

	var v {{.TypeName}}
	err = json.Unmarshal(jsonData, &v)
	if err == nil {
		*s = {{.Name}}(v)
		return nil
	}

	return errors.New("failed to unmarshal {{.Name}}")
}`))

func (t *YacIt) createProtoEnum(typ reflect.Type) {
	name := typ.Name()
	if _, ok := t.types[name]; ok {
		return
	}

	sourceType := typ.String()
	var res strings.Builder
	// type <name> <underlying type>
	res.WriteString("type ")
	res.WriteString(name)
	res.WriteByte(' ')
	res.WriteString(sourceType)
	res.WriteByte('\n')

	tmplData := struct {
		Name     string
		TypeName string
	}{
		Name:     name,
		TypeName: sourceType,
	}
	if err := protoEnumTemplate.Execute(&res, tmplData); err != nil {
		panic(err)
	}
	res.WriteByte('\n')

	t.types[name] = res.String()
}

type ProtoEnum interface {
	Descriptor() protoreflect.EnumDescriptor
	Type() protoreflect.EnumType
	Number() protoreflect.EnumNumber
}

type YamlUnmarshaler interface {
	UnmarshalYAML(node *yaml.Node) error
}

func hasYamlUnmarshaler(typ reflect.Type) bool {
	return reflect.PtrTo(typ).Implements(reflect.TypeOf((*YamlUnmarshaler)(nil)).Elem())
}

func shouldGenerateTypeForEnum(typ reflect.Type) bool {
	return isProtoEnum(typ) && !hasYamlUnmarshaler(typ)
}

// unlike typ.String() we strip the namespace from all types
func printType(typ reflect.Type) string {
	switch typ.Kind() {
	case reflect.Slice:
		return "[]" + printType(typ.Elem())
	case reflect.Pointer:
		return "*" + printType(typ.Elem())
	case reflect.Map:
		return "map[" + typ.Key().String() + "]" + printType(typ.Elem())
	case reflect.Array:
		// TODO: technically we need the number here...
		return "[]" + printType(typ.Elem())
	case reflect.Struct:
		return typ.Name()
	default:
		if shouldGenerateTypeForEnum(typ) {
			return typ.Name()
		}
		return typ.String()
	}
}

func baseType(typ reflect.Type) reflect.Type {
	for {
		switch typ.Kind() {
		case reflect.Slice, reflect.Pointer, reflect.Map, reflect.Array:
			typ = typ.Elem()
		default:
			return typ
		}
	}
}

func addYamlTag(s reflect.StructTag) reflect.StructTag {
	j := s.Get("json")
	if j == "" {
		return s
	}

	res := string(s) + " yaml:" + fmt.Sprintf("%#v", j)
	return reflect.StructTag(res)
}
