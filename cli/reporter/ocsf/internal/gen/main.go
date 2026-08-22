// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Command gen generates the OCSF Go types cnspec emits, from the compiled OCSF
// schemas in ../../schemas and the attribute list in ../../gen.yaml.
//
// Run it with `go generate ./cli/reporter/ocsf/...`.
//
// Everything about a field except whether it is present comes from the schema:
// its OCSF name, its Go type, whether it is optional, its enum values and their
// captions, and its documentation. That is the point of generating: a hand
// written struct can disagree with the schema, and the disagreement only shows
// up when a data lake rejects the events.
package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"sigs.k8s.io/yaml"
)

type spec struct {
	Classes   map[string]classSpec `json:"classes"`
	Objects   map[string][]string  `json:"objects"`
	Overrides map[string]string    `json:"overrides"`
}

type classSpec struct {
	Go         string   `json:"go"`
	Attributes []string `json:"attributes"`
}

type schema struct {
	Version string            `json:"version"`
	Classes map[string]entity `json:"classes"`
	Objects map[string]entity `json:"objects"`
}

type entity struct {
	Name         string          `json:"name"`
	Caption      string          `json:"caption"`
	Description  string          `json:"description"`
	UID          int             `json:"uid"`
	CategoryUID  int             `json:"category_uid"`
	CategoryName string          `json:"category_name"`
	Profiles     []string        `json:"profiles"`
	Attributes   map[string]attr `json:"attributes"`
}

type attr struct {
	Caption     string               `json:"caption"`
	Description string               `json:"description"`
	Type        string               `json:"type"`
	ObjectType  string               `json:"object_type"`
	IsArray     bool                 `json:"is_array"`
	Requirement string               `json:"requirement"`
	Profiles    []string             `json:"profiles"`
	Sibling     string               `json:"sibling"`
	Enum        map[string]enumValue `json:"enum"`
}

type enumValue struct {
	Caption     string `json:"caption"`
	Description string `json:"description"`
}

// field is one resolved struct field, with the versions it exists in.
type field struct {
	OCSFName string
	GoName   string
	GoType   string
	Optional bool
	IsArray  bool
	IsMap    bool
	Doc      string
	Desc     string
	InAll    bool
	Versions []string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
}

func run() error {
	out := flag.String("o", "", "directory to write the generated files to (default: the ocsf package)")
	flag.Parse()

	root, err := packageRoot()
	if err != nil {
		return err
	}
	dest := root
	if *out != "" {
		dest = *out
	}

	var sp spec
	raw, err := os.ReadFile(filepath.Join(root, "gen.yaml"))
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(raw, &sp); err != nil {
		return fmt.Errorf("gen.yaml: %w", err)
	}

	// The schemas directory is the version list. Keeping a second copy here
	// would be one more thing to hold in sync with ocsf.SupportedVersions.
	paths, err := filepath.Glob(filepath.Join(root, "schemas", "schema-*.json.gz"))
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("no compiled schemas in %s", filepath.Join(root, "schemas"))
	}
	sort.Strings(paths)

	schemas := make([]schema, 0, len(paths))
	for _, path := range paths {
		s, err := loadSchema(path)
		if err != nil {
			return err
		}
		want := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(path), "schema-"), ".json.gz")
		if s.Version != want {
			return fmt.Errorf("%s declares version %q", filepath.Base(path), s.Version)
		}
		schemas = append(schemas, s)
	}

	g := &generator{spec: sp, schemas: schemas}
	types, err := g.types()
	if err != nil {
		return err
	}
	enums, err := g.enums()
	if err != nil {
		return err
	}

	for name, content := range map[string][]byte{"types.gen.go": types, "enums.gen.go": enums} {
		formatted, err := format.Source(content)
		if err != nil {
			// Write the unformatted source so the syntax error is readable.
			_ = os.WriteFile(filepath.Join(dest, name+".broken"), content, 0o644)
			return fmt.Errorf("%s: %w", name, err)
		}
		if err := os.WriteFile(filepath.Join(dest, name), formatted, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// packageRoot is the ocsf package directory, two levels up from internal/gen.
func packageRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	// go generate runs with the working directory of the file that declares it.
	for _, candidate := range []string{wd, filepath.Join(wd, "..", ".."), filepath.Join(wd, "..", "..", "..")} {
		if _, err := os.Stat(filepath.Join(candidate, "gen.yaml")); err == nil {
			return filepath.Clean(candidate), nil
		}
	}
	return "", fmt.Errorf("cannot find gen.yaml from %s", wd)
}

func loadSchema(path string) (schema, error) {
	var s schema
	f, err := os.Open(path)
	if err != nil {
		return s, err
	}
	defer f.Close() //nolint: errcheck

	gz, err := gzip.NewReader(f)
	if err != nil {
		return s, fmt.Errorf("%s: %w", path, err)
	}
	defer gz.Close() //nolint: errcheck

	buf := bytes.Buffer{}
	if _, err := buf.ReadFrom(gz); err != nil {
		return s, fmt.Errorf("%s: %w", path, err)
	}
	if err := yaml.Unmarshal(buf.Bytes(), &s); err != nil {
		return s, fmt.Errorf("%s: %w", path, err)
	}
	return s, nil
}

type generator struct {
	spec    spec
	schemas []schema
}

// lookupClass finds a class attribute across the supported versions, returning
// its definition and the versions that have it.
func (g *generator) lookupClass(class, attribute string) (attr, []string) {
	return g.lookup(func(s schema) (entity, bool) { e, ok := s.Classes[class]; return e, ok }, attribute)
}

func (g *generator) lookupObject(object, attribute string) (attr, []string) {
	return g.lookup(func(s schema) (entity, bool) { e, ok := s.Objects[object]; return e, ok }, attribute)
}

func (g *generator) lookup(pick func(schema) (entity, bool), attribute string) (attr, []string) {
	var found attr
	var in []string
	for _, s := range g.schemas {
		e, ok := pick(s)
		if !ok {
			continue
		}
		a, ok := e.Attributes[attribute]
		if !ok {
			continue
		}
		if len(in) == 0 {
			found = a
		}
		in = append(in, s.Version)
	}
	return found, in
}

func (g *generator) entityIn(kind, name string) (entity, []string) {
	var found entity
	var in []string
	for _, s := range g.schemas {
		set := s.Objects
		if kind == "class" {
			set = s.Classes
		}
		e, ok := set[name]
		if !ok {
			continue
		}
		if len(in) == 0 {
			found = e
		}
		in = append(in, s.Version)
	}
	return found, in
}

func (g *generator) types() ([]byte, error) {
	b := &strings.Builder{}
	writeHeader(b, "types.gen.go")

	b.WriteString("package ocsf\n\n")

	// Classes, in the order the spec lists them.
	for _, name := range sortedKeys(g.spec.Classes) {
		cs := g.spec.Classes[name]
		class, in := g.entityIn("class", name)
		if len(in) == 0 {
			return nil, fmt.Errorf("class %q is in no supported schema version", name)
		}

		fields := make([]field, 0, len(cs.Attributes))
		for _, attribute := range cs.Attributes {
			a, versions := g.lookupClass(name, attribute)
			if len(versions) == 0 {
				return nil, fmt.Errorf("class %q has no attribute %q in any supported version", name, attribute)
			}
			f, err := g.field(attribute, a, versions)
			if err != nil {
				return nil, fmt.Errorf("%s.%s: %w", name, attribute, err)
			}
			fields = append(fields, f)
		}

		writeDoc(b, cs.Go+" is OCSF class "+itoa(class.UID)+", "+class.Caption+".", class.Description, in, versionList(g))
		b.WriteString("type " + cs.Go + " struct {\n")
		writeFields(b, fields)
		b.WriteString("}\n\n")

		writeConstructor(b, cs.Go, fields)
	}

	// Objects.
	for _, name := range sortedKeys(g.spec.Objects) {
		object, in := g.entityIn("object", name)
		if len(in) == 0 {
			return nil, fmt.Errorf("object %q is in no supported schema version", name)
		}

		fields := make([]field, 0, len(g.spec.Objects[name]))
		for _, attribute := range g.spec.Objects[name] {
			a, versions := g.lookupObject(name, attribute)
			if len(versions) == 0 {
				return nil, fmt.Errorf("object %q has no attribute %q in any supported version", name, attribute)
			}
			f, err := g.field(attribute, a, versions)
			if err != nil {
				return nil, fmt.Errorf("%s.%s: %w", name, attribute, err)
			}
			fields = append(fields, f)
		}

		writeDoc(b, goName(name)+" is the OCSF "+object.Caption+" object.", object.Description, in, versionList(g))
		b.WriteString("type " + goName(name) + " struct {\n")
		writeFields(b, fields)
		b.WriteString("}\n\n")
	}

	return []byte(b.String()), nil
}

// field resolves one attribute into a Go struct field.
func (g *generator) field(name string, a attr, in []string) (field, error) {
	f := field{
		OCSFName: name,
		GoName:   goName(name),
		IsArray:  a.IsArray,
		// An attribute that belongs to a profile is only present when cnspec
		// declares that profile, so it is optional in Go whatever the schema
		// says: the schema's requirement assumes the profile is applied.
		Optional: a.Requirement != "required" || len(a.Profiles) > 0,
		Doc:      a.Caption,
		Desc:     a.Description,
		InAll:    len(in) == len(g.schemas),
		Versions: in,
	}
	if len(a.Profiles) > 0 {
		f.Doc += " (OCSF " + strings.Join(a.Profiles, ", ") + " profile)"
	}

	if override, ok := g.spec.Overrides[name]; ok {
		f.GoType = override
		f.IsMap = strings.HasPrefix(override, "map[")
		return f, nil
	}

	switch a.Type {
	case "string_t", "datetime_t", "email_t", "file_hash_t", "file_name_t", "hostname_t",
		"ip_t", "mac_t", "process_name_t", "subnet_t", "url_t", "username_t", "uuid_t":
		f.GoType = "string"
	case "integer_t", "port_t":
		f.GoType = "int"
	case "long_t", "timestamp_t":
		f.GoType = "int64"
	case "float_t":
		f.GoType = "float64"
	case "boolean_t":
		f.GoType = "bool"
	case "object_t":
		if a.ObjectType == "" {
			return f, fmt.Errorf("object attribute without an object_type")
		}
		if _, ok := g.spec.Objects[a.ObjectType]; !ok {
			return f, fmt.Errorf("object type %q is not listed in gen.yaml", a.ObjectType)
		}
		f.GoType = goName(a.ObjectType)
		if !a.IsArray && f.Optional {
			// An optional nested object is a pointer, so an absent one is null
			// rather than an empty object. A required one is always written.
			f.GoType = "*" + f.GoType
		}
	default:
		return f, fmt.Errorf("unsupported OCSF type %q", a.Type)
	}

	if a.IsArray {
		f.GoType = "[]" + f.GoType
	}
	return f, nil
}

// writeConstructor emits a constructor that fills in the classification
// attributes of a class. They are pure schema: the category and class are fixed,
// the activity is chosen by the caller, and type_uid is the two combined. Having
// the generator derive them means an event cannot claim to be a class it is not.
func writeConstructor(b *strings.Builder, goClass string, fields []field) {
	has := map[string]bool{}
	for _, f := range fields {
		has[f.OCSFName] = true
	}
	for _, required := range []string{"activity_id", "category_uid", "class_uid", "type_uid"} {
		if !has[required] {
			return
		}
	}

	b.WriteString("// New" + goClass + " returns a " + goClass + " with its classification\n" +
		"// attributes filled in from the schema. Pass one of the " + goClass + "Activity\n" +
		"// constants.\n")
	b.WriteString("func New" + goClass + "(activityID int) " + goClass + " {\n")
	b.WriteString("\ttypeUID := ClassUID" + goClass + "*100 + activityID\n")
	b.WriteString("\treturn " + goClass + "{\n")
	b.WriteString("\t\tActivityID: activityID,\n")
	if has["activity_name"] {
		b.WriteString("\t\tActivityName: " + goClass + "ActivityName(activityID),\n")
	}
	b.WriteString("\t\tCategoryUID: CategoryUID" + goClass + ",\n")
	if has["category_name"] {
		b.WriteString("\t\tCategoryName: CategoryName" + goClass + ",\n")
	}
	b.WriteString("\t\tClassUID: ClassUID" + goClass + ",\n")
	if has["class_name"] {
		b.WriteString("\t\tClassName: ClassName" + goClass + ",\n")
	}
	b.WriteString("\t\tTypeUID: int64(typeUID),\n")
	if has["type_name"] {
		b.WriteString("\t\tTypeName: " + goClass + "TypeUIDName(typeUID),\n")
	}
	b.WriteString("\t}\n}\n\n")
}

func writeFields(b *strings.Builder, fields []field) {
	for i, f := range fields {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("\t// " + f.Doc + "\n")
		if desc := oneLine(f.Desc); desc != "" && desc != f.Doc {
			b.WriteString("\t//\n")
			for _, line := range wrap(desc, 72) {
				b.WriteString("\t// " + line + "\n")
			}
		}
		if !f.InAll {
			b.WriteString("\t//\n\t// Only in OCSF " + strings.Join(f.Versions, ", ") + "; empty otherwise.\n")
		}

		jsonTag := f.OCSFName
		if f.Optional || f.IsArray || f.IsMap {
			jsonTag += ",omitempty"
		}

		parquetTag := f.OCSFName
		switch {
		case f.IsArray:
			parquetTag += ",list"
		case f.IsMap:
			// maps are written as a parquet MAP; they are never null
		case f.Optional:
			parquetTag += ",optional"
		}

		b.WriteString("\t" + f.GoName + " " + f.GoType +
			" `json:\"" + jsonTag + "\" parquet:\"" + parquetTag + "\"`\n")
	}
}

// enums generates the identifier constants and caption lookups of every enum
// attribute in the spec, so no enum value or label is written by hand.
func (g *generator) enums() ([]byte, error) {
	b := &strings.Builder{}
	writeHeader(b, "enums.gen.go")
	b.WriteString("package ocsf\n\n")

	// Class identity: uid, caption, category.
	b.WriteString("// Event classes cnspec emits.\nconst (\n")
	for _, name := range sortedKeys(g.spec.Classes) {
		cs := g.spec.Classes[name]
		class, _ := g.entityIn("class", name)
		b.WriteString("\t// " + cs.Go + " identity, from the OCSF schema.\n")
		b.WriteString("\tClassUID" + cs.Go + " = " + itoa(class.UID) + "\n")
		b.WriteString("\tClassName" + cs.Go + " = " + quote(class.Caption) + "\n")
		b.WriteString("\tClass" + cs.Go + " = " + quote(class.Name) + "\n")
		b.WriteString("\tCategoryUID" + cs.Go + " = " + itoa(class.CategoryUID) + "\n")
		b.WriteString("\tCategoryName" + cs.Go + " = " + quote(class.CategoryName) + "\n\n")
	}
	b.WriteString(")\n\n")

	// Profiles, so a declared profile is never a loose string.
	profiles := map[string]bool{}
	for name := range g.spec.Classes {
		class, _ := g.entityIn("class", name)
		for _, p := range class.Profiles {
			profiles[p] = true
		}
	}
	b.WriteString("// OCSF profiles. An attribute that belongs to a profile is only valid on an\n" +
		"// event whose metadata declares the profile.\nconst (\n")
	for _, p := range sortedKeys(profiles) {
		if strings.Contains(p, "/") {
			continue // extension profiles, e.g. linux/linux_users
		}
		b.WriteString("\tProfile" + goName(p) + " = " + quote(p) + "\n")
	}
	b.WriteString(")\n\n")

	// Enum constants and caption lookups.
	for _, e := range g.enumAttributes() {
		b.WriteString("// " + e.prefix + " values of " + e.source + ", from the OCSF schema.\nconst (\n")
		for _, id := range e.sortedIDs() {
			value := e.values[id]
			if value.Description != "" {
				b.WriteString("\t// " + oneLine(value.Description) + "\n")
			}
			// type_uid captions repeat the class ("Compliance Finding: Create"),
			// which the prefix already carries.
			caption := value.Caption
			if _, rest, found := strings.Cut(caption, ": "); found {
				caption = rest
			}
			b.WriteString("\t" + e.prefix + goName(caption) + " = " + id + "\n")
		}
		b.WriteString(")\n\n")

		b.WriteString("// " + e.prefix + "Name is the OCSF caption of a " + e.source + " value. It is what the\n" +
			"// string sibling of the identifier has to carry.\n")
		b.WriteString("func " + e.prefix + "Name(id int) string {\n\tswitch id {\n")
		for _, id := range e.sortedIDs() {
			b.WriteString("\tcase " + id + ":\n\t\treturn " + quote(e.values[id].Caption) + "\n")
		}
		b.WriteString("\t}\n\treturn \"\"\n}\n\n")
	}

	return []byte(b.String()), nil
}

type enumDef struct {
	prefix string
	source string
	values map[string]enumValue
}

func (e enumDef) sortedIDs() []string {
	ids := make([]string, 0, len(e.values))
	for id := range e.values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return atoi(ids[i]) < atoi(ids[j]) })
	return ids
}

// enumAttributes collects every enum in the spec. An enum that is identical
// everywhere it appears gets a short prefix (Severity, Status); one that differs
// per class is prefixed with the class, because activity_id means something
// different on a finding than on a discovery event.
func (g *generator) enumAttributes() []enumDef {
	var all []enumOccurrence

	for _, name := range sortedKeys(g.spec.Classes) {
		cs := g.spec.Classes[name]
		for _, attribute := range cs.Attributes {
			if identityAttributes[attribute] {
				// class_uid and category_uid are identity, not a choice; they are
				// already emitted as ClassUID<Class> and CategoryUID<Class>.
				continue
			}
			if a, in := g.lookupClass(name, attribute); len(in) > 0 && len(a.Enum) > 0 {
				all = append(all, enumOccurrence{owner: "class", goName: cs.Go, attr: attribute, values: a.Enum})
			}
		}
	}
	for _, name := range sortedKeys(g.spec.Objects) {
		for _, attribute := range g.spec.Objects[name] {
			if a, in := g.lookupObject(name, attribute); len(in) > 0 && len(a.Enum) > 0 {
				all = append(all, enumOccurrence{owner: "object", goName: goName(name), attr: attribute, values: a.Enum})
			}
		}
	}

	// group class enums by attribute to see whether they agree
	byAttr := map[string][]enumOccurrence{}
	for _, o := range all {
		if o.owner == "class" {
			byAttr[o.attr] = append(byAttr[o.attr], o)
		}
	}

	var res []enumDef
	seen := map[string]bool{}
	for _, o := range all {
		var prefix, source string
		if o.owner == "class" {
			group := byAttr[o.attr]
			if sameEnum(group) {
				prefix = goName(strings.TrimSuffix(o.attr, "_id"))
				source = o.attr
			} else {
				prefix = o.goName + goName(strings.TrimSuffix(o.attr, "_id"))
				source = o.goName + "." + o.attr
			}
		} else {
			prefix = o.goName + goName(strings.TrimSuffix(o.attr, "_id"))
			source = o.goName + "." + o.attr
		}
		if seen[prefix] {
			continue
		}
		seen[prefix] = true
		res = append(res, enumDef{prefix: prefix, source: source, values: o.values})
	}
	sort.Slice(res, func(i, j int) bool { return res[i].prefix < res[j].prefix })
	return res
}

// identityAttributes are enums that carry the class's own identity rather than a
// choice, so they are emitted as plain identity constants instead.
var identityAttributes = map[string]bool{"class_uid": true, "category_uid": true}

// enumOccurrence is one place an enum attribute shows up in the spec.
type enumOccurrence struct {
	owner  string
	goName string
	attr   string
	values map[string]enumValue
}

// sameEnum reports whether every occurrence of an attribute carries the same
// enum. activity_id does not: on a finding it is Create/Update/Close, on a
// discovery event it is Log/Collect, so those need class-scoped names.
func sameEnum(group []enumOccurrence) bool {
	for _, o := range group[1:] {
		if len(o.values) != len(group[0].values) {
			return false
		}
		for id, v := range o.values {
			if other, ok := group[0].values[id]; !ok || other.Caption != v.Caption {
				return false
			}
		}
	}
	return true
}

func writeHeader(b *strings.Builder, name string) {
	b.WriteString("// Copyright Mondoo, Inc. 2024, 2026\n// SPDX-License-Identifier: BUSL-1.1\n\n")
	b.WriteString("// Code generated by cli/reporter/ocsf/internal/gen from the compiled OCSF\n" +
		"// schemas in schemas/ and the attribute list in gen.yaml. DO NOT EDIT.\n\n")
}

func writeDoc(b *strings.Builder, summary, description string, in, all []string) {
	b.WriteString("// " + summary + "\n")
	if description != "" {
		b.WriteString("//\n")
		for _, line := range wrap(oneLine(description), 76) {
			b.WriteString("// " + line + "\n")
		}
	}
	if len(in) != len(all) {
		b.WriteString("//\n// Only in OCSF " + strings.Join(in, ", ") + ".\n")
	}
}

func versionList(g *generator) []string {
	res := make([]string, 0, len(g.schemas))
	for _, s := range g.schemas {
		res = append(res, s.Version)
	}
	return res
}

var (
	tagRe        = regexp.MustCompile(`<[^>]+>`)
	whitespaceRe = regexp.MustCompile(`\s+`)
	wordRe       = regexp.MustCompile(`[A-Za-z0-9]+`)
)

// oneLine turns a schema description, which carries HTML, into plain prose.
func oneLine(s string) string {
	s = strings.ReplaceAll(s, "<p>", " ")
	s = tagRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	return strings.TrimSpace(whitespaceRe.ReplaceAllString(s, " "))
}

func wrap(s string, width int) []string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(s) {
		if line != "" && len(line)+1+len(word) > width {
			lines = append(lines, line)
			line = ""
		}
		if line != "" {
			line += " "
		}
		line += word
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

// initialisms keep generated names idiomatic: Uid reads wrong where UID does not.
var initialisms = map[string]string{
	"api": "API", "cpu": "CPU", "cve": "CVE", "cvss": "CVSS", "cwe": "CWE",
	"id": "ID", "ip": "IP", "json": "JSON", "os": "OS", "purl": "PURL",
	"ids": "IDS", "iot": "IOT", "ips": "IPS",
	"tlp": "TLP", "uid": "UID", "url": "URL", "uuid": "UUID",
}

func goName(s string) string {
	b := strings.Builder{}
	for _, word := range wordRe.FindAllString(s, -1) {
		lower := strings.ToLower(word)
		if replacement, ok := initialisms[lower]; ok {
			b.WriteString(replacement)
			continue
		}
		// Captions carry their own capitalization: AIX stays AIX and macOS
		// becomes MacOS rather than Macos.
		if word == strings.ToUpper(word) || strings.ToLower(word[1:]) != word[1:] {
			b.WriteString(strings.ToUpper(word[:1]) + word[1:])
			continue
		}
		b.WriteString(strings.ToUpper(lower[:1]) + lower[1:])
	}
	return b.String()
}

func sortedKeys[V any](m map[string]V) []string {
	res := make([]string, 0, len(m))
	for k := range m {
		res = append(res, k)
	}
	slices.Sort(res)
	return res
}

func quote(s string) string { return "\"" + strings.ReplaceAll(s, "\"", "\\\"") + "\"" }

func itoa(i int) string { return fmt.Sprintf("%d", i) }

func atoi(s string) int {
	var i int
	_, _ = fmt.Sscanf(s, "%d", &i)
	return i
}
