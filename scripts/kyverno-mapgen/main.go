// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

const (
	catalogVersion = 2

	defaultSourceName = "kyverno/policies"
	defaultSourceURL  = "https://github.com/kyverno/policies"

	defaultMondooPolicyUid      = "mondoo-kubernetes-security"
	defaultMondooCheckMrnPrefix = "//policy.api.mondoo.app/policies/mondoo-kubernetes-security/checks/"

	defaultMappingSource    = "official-kyverno-policy-catalog"
	defaultMappedConfidence = "high"
	defaultUnmappedStatus   = "unmapped"
	defaultReason           = "Generated from the pinned official Kyverno policies catalog. High-confidence entries are exact or narrow semantic matches; partial entries require review before automated exception mirroring."
	defaultCatalogPath      = "content/kyverno/official-policy-mappings.yaml"
)

var supportedPolicyKinds = map[string]struct{}{
	"ClusterPolicy":                   {},
	"Policy":                          {},
	"ValidatingPolicy":                {},
	"NamespacedValidatingPolicy":      {},
	"ImageValidatingPolicy":           {},
	"NamespacedImageValidatingPolicy": {},
	"MutatingPolicy":                  {},
	"GeneratingPolicy":                {},
	"DeletingPolicy":                  {},
	"NamespacedDeletingPolicy":        {},
}

type catalog struct {
	Version  int             `yaml:"version" json:"version"`
	Source   catalogSource   `yaml:"source" json:"source"`
	Mondoo   mondooDefaults  `yaml:"mondoo,omitempty" json:"mondoo,omitempty"`
	Defaults catalogDefaults `yaml:"defaults,omitempty" json:"defaults,omitempty"`
	Policies []policyEntry   `yaml:"policies" json:"policies"`
	Mappings []mappingEntry  `yaml:"mappings,omitempty" json:"mappings,omitempty"`
}

type catalogSource struct {
	Name                string `yaml:"name,omitempty" json:"name,omitempty"`
	URL                 string `yaml:"url,omitempty" json:"url,omitempty"`
	Ref                 string `yaml:"ref" json:"ref"`
	PolicyResourceCount int    `yaml:"policyResourceCount" json:"policyResourceCount"`
	UniquePolicyCount   int    `yaml:"uniquePolicyCount" json:"uniquePolicyCount"`
}

type mondooDefaults struct {
	PolicyUID      string `yaml:"policyUid,omitempty" json:"policyUid,omitempty"`
	CheckMRNPrefix string `yaml:"checkMrnPrefix,omitempty" json:"checkMrnPrefix,omitempty"`
}

type catalogDefaults struct {
	Source           string `yaml:"source,omitempty" json:"source,omitempty"`
	MappedConfidence string `yaml:"mappedConfidence,omitempty" json:"mappedConfidence,omitempty"`
	UnmappedStatus   string `yaml:"unmappedStatus,omitempty" json:"unmappedStatus,omitempty"`
	Reason           string `yaml:"reason,omitempty" json:"reason,omitempty"`
}

type policyEntry struct {
	KyvernoPolicy string            `yaml:"kyvernoPolicy" json:"kyvernoPolicy"`
	MappingStatus string            `yaml:"mappingStatus" json:"mappingStatus"`
	MappingRefs   []string          `yaml:"mappingRefs" json:"mappingRefs"`
	KyvernoRules  []string          `yaml:"kyvernoRules,omitempty" json:"kyvernoRules,omitempty"`
	Upstream      upstreamPolicySet `yaml:"upstream" json:"upstream"`
}

type upstreamPolicySet struct {
	Titles          []string         `yaml:"titles,omitempty" json:"titles,omitempty"`
	Categories      []string         `yaml:"categories,omitempty" json:"categories,omitempty"`
	PolicyResources []policyResource `yaml:"policyResources" json:"policyResources"`
}

type policyResource struct {
	Path       string `yaml:"path" json:"path"`
	APIVersion string `yaml:"apiVersion" json:"apiVersion"`
	Kind       string `yaml:"kind" json:"kind"`
}

type mappingEntry struct {
	UID             string   `yaml:"uid" json:"uid"`
	KyvernoPolicy   string   `yaml:"kyvernoPolicy" json:"kyvernoPolicy"`
	KyvernoRules    []string `yaml:"kyvernoRules" json:"kyvernoRules"`
	MondooPolicyUID string   `yaml:"mondooPolicyUid,omitempty" json:"mondooPolicyUid,omitempty"`
	MondooChecks    []string `yaml:"mondooChecks" json:"mondooChecks"`
	Source          string   `yaml:"source,omitempty" json:"source,omitempty"`
	Confidence      string   `yaml:"confidence" json:"confidence"`
	MappingStatus   string   `yaml:"mappingStatus" json:"mappingStatus"`
	Reason          string   `yaml:"reason,omitempty" json:"reason,omitempty"`
}

type discoveredPolicy struct {
	name       string
	rules      map[string]struct{}
	titles     map[string]struct{}
	categories map[string]struct{}
	resources  []policyResource
}

func main() {
	var upstreamDir string
	var sourceRef string
	var existingPath string
	var outPath string
	var check bool

	flag.StringVar(&upstreamDir, "kyverno-policies", "", "Path to a local checkout of github.com/kyverno/policies")
	flag.StringVar(&sourceRef, "source-ref", "", "Pinned kyverno/policies git ref for the generated catalog")
	flag.StringVar(&existingPath, "existing", defaultCatalogPath, "Existing catalog to preserve reviewed mappings from; defaults to the in-repo output catalog for in-place updates")
	flag.StringVar(&outPath, "out", defaultCatalogPath, "Output catalog path; defaults to the in-repo Kyverno mapping catalog")
	flag.BoolVar(&check, "check", false, "Fail if the generated catalog differs from the output file")
	flag.Parse()

	if upstreamDir == "" {
		fatalf("--kyverno-policies is required")
	}
	if sourceRef == "" {
		fatalf("--source-ref is required")
	}

	existing, err := readCatalog(existingPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fatalf("failed to read existing catalog: %v", err)
	}

	generated, err := buildCatalog(upstreamDir, sourceRef, existing)
	if err != nil {
		fatalf("failed to build catalog: %v", err)
	}
	out, err := marshalCatalog(generated)
	if err != nil {
		fatalf("failed to marshal catalog: %v", err)
	}

	if check {
		current, err := readCatalog(outPath)
		if err != nil {
			fatalf("failed to read output catalog: %v", err)
		}
		if !catalogSemanticallyEqual(current, generated) {
			fatalf("%s is not up to date; rerun kyverno-mapgen", outPath)
		}
		return
	}

	if err := os.WriteFile(outPath, out, 0o644); err != nil {
		fatalf("failed to write output catalog: %v", err)
	}
}

func readCatalog(path string) (*catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c catalog
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func buildCatalog(upstreamDir string, sourceRef string, existing *catalog) (*catalog, error) {
	policies, resourceCount, err := discoverPolicies(upstreamDir)
	if err != nil {
		return nil, err
	}

	c := &catalog{
		Version: catalogVersion,
		Source: catalogSource{
			Name:                defaultSourceName,
			URL:                 defaultSourceURL,
			Ref:                 sourceRef,
			PolicyResourceCount: resourceCount,
			UniquePolicyCount:   len(policies),
		},
		Mondoo: mondooDefaults{
			PolicyUID:      defaultMondooPolicyUid,
			CheckMRNPrefix: defaultMondooCheckMrnPrefix,
		},
		Defaults: catalogDefaults{
			Source:           defaultMappingSource,
			MappedConfidence: defaultMappedConfidence,
			UnmappedStatus:   defaultUnmappedStatus,
			Reason:           defaultReason,
		},
		Mappings: reviewedMappings(existing),
	}
	if existing != nil {
		if existing.Source.Name != "" {
			c.Source.Name = existing.Source.Name
		}
		if existing.Source.URL != "" {
			c.Source.URL = existing.Source.URL
		}
		if existing.Mondoo.PolicyUID != "" {
			c.Mondoo.PolicyUID = existing.Mondoo.PolicyUID
		}
		if existing.Mondoo.CheckMRNPrefix != "" {
			c.Mondoo.CheckMRNPrefix = existing.Mondoo.CheckMRNPrefix
		}
		if existing.Defaults.Source != "" {
			c.Defaults.Source = existing.Defaults.Source
		}
		if existing.Defaults.MappedConfidence != "" {
			c.Defaults.MappedConfidence = existing.Defaults.MappedConfidence
		}
		if existing.Defaults.UnmappedStatus != "" {
			c.Defaults.UnmappedStatus = existing.Defaults.UnmappedStatus
		}
		if existing.Defaults.Reason != "" {
			c.Defaults.Reason = existing.Defaults.Reason
		}
	}

	mappingsByPolicy := map[string][]mappingEntry{}
	for _, mapping := range c.Mappings {
		mappingsByPolicy[mapping.KyvernoPolicy] = append(mappingsByPolicy[mapping.KyvernoPolicy], mapping)
	}

	names := make([]string, 0, len(policies))
	for name := range policies {
		names = append(names, name)
	}
	sort.Strings(names)

	c.Policies = make([]policyEntry, 0, len(names))
	for _, name := range names {
		policy := policies[name]
		mappings := mappingsByPolicy[name]
		refs := make([]string, 0, len(mappings))
		status := defaultUnmappedStatus
		if len(mappings) > 0 {
			status = "mapped"
			mappedRules := map[string]struct{}{}
			for _, mapping := range mappings {
				refs = append(refs, mapping.UID)
				for _, rule := range mapping.KyvernoRules {
					mappedRules[rule] = struct{}{}
				}
				if mapping.MappingStatus == "partial" {
					status = "partial"
				}
			}
			for rule := range policy.rules {
				if _, ok := mappedRules[rule]; !ok {
					status = "partial"
					break
				}
			}
			sort.Strings(refs)
		}
		c.Policies = append(c.Policies, policyEntry{
			KyvernoPolicy: name,
			MappingStatus: status,
			MappingRefs:   refs,
			KyvernoRules:  sortedKeys(policy.rules),
			Upstream: upstreamPolicySet{
				Titles:          sortedKeys(policy.titles),
				Categories:      sortedKeys(policy.categories),
				PolicyResources: sortedResources(policy.resources),
			},
		})
	}

	return c, nil
}

func reviewedMappings(existing *catalog) []mappingEntry {
	if existing == nil || len(existing.Mappings) == 0 {
		return nil
	}
	mappings := append([]mappingEntry(nil), existing.Mappings...)
	sort.SliceStable(mappings, func(i, j int) bool {
		return mappings[i].UID < mappings[j].UID
	})
	return mappings
}

func discoverPolicies(root string) (map[string]*discoveredPolicy, int, error) {
	policies := map[string]*discoveredPolicy{}
	resourceCount := 0

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".github", ".chainsaw-test", ".kyverno-test":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		if !isYAML(path) {
			return nil
		}
		resources, err := parsePolicyResources(root, path)
		if err != nil {
			return err
		}
		for _, resource := range resources {
			policy := policies[resource.name]
			if policy == nil {
				policy = &discoveredPolicy{
					name:       resource.name,
					rules:      map[string]struct{}{},
					titles:     map[string]struct{}{},
					categories: map[string]struct{}{},
				}
				policies[resource.name] = policy
			}
			for _, rule := range resource.rules {
				policy.rules[rule] = struct{}{}
			}
			for _, title := range resource.titles {
				policy.titles[title] = struct{}{}
			}
			for _, category := range resource.categories {
				policy.categories[category] = struct{}{}
			}
			policy.resources = append(policy.resources, resource.policyResource)
			resourceCount++
		}
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	return policies, resourceCount, nil
}

type parsedPolicyResource struct {
	name           string
	rules          []string
	titles         []string
	categories     []string
	policyResource policyResource
}

func parsePolicyResources(root string, path string) ([]parsedPolicyResource, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	rel, err := filepath.Rel(root, path)
	if err != nil {
		return nil, err
	}
	rel = filepath.ToSlash(rel)

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	out := []parsedPolicyResource{}
	for {
		doc := map[string]any{}
		err := decoder.Decode(&doc)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", rel, err)
		}
		if len(doc) == 0 {
			continue
		}
		apiVersion := stringFromPath(doc, "apiVersion")
		kind := stringFromPath(doc, "kind")
		if !isSupportedPolicy(apiVersion, kind) {
			continue
		}
		name := stringFromPath(doc, "metadata", "name")
		if name == "" {
			return nil, fmt.Errorf("%s: %s has no metadata.name", rel, kind)
		}
		annotations := mapFromPath(doc, "metadata", "annotations")
		out = append(out, parsedPolicyResource{
			name:       name,
			rules:      extractRules(doc),
			titles:     annotationValueList(stringFromMap(annotations, "policies.kyverno.io/title")),
			categories: annotationValueList(stringFromMap(annotations, "policies.kyverno.io/category")),
			policyResource: policyResource{
				Path:       rel,
				APIVersion: apiVersion,
				Kind:       kind,
			},
		})
	}
	return out, nil
}

func extractRules(doc map[string]any) []string {
	ruleNames := map[string]struct{}{}
	for _, path := range [][]string{
		{"spec", "rules"},
		{"spec", "validations"},
		{"spec", "mutations"},
		{"spec", "generate"},
	} {
		for _, item := range mapsFromPath(doc, path...) {
			if name := stringFromMap(item, "name"); name != "" {
				ruleNames[name] = struct{}{}
			}
		}
	}
	return sortedKeys(ruleNames)
}

func isSupportedPolicy(apiVersion string, kind string) bool {
	if !strings.Contains(apiVersion, "kyverno.io/") {
		return false
	}
	_, ok := supportedPolicyKinds[kind]
	return ok
}

func isYAML(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func marshalCatalog(c *catalog) ([]byte, error) {
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.CompactSeqIndent()
	encoder.SetIndent(2)
	if err := encoder.Encode(c); err != nil {
		_ = encoder.Close()
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return append([]byte("# Copyright Mondoo, Inc. 2026\n# SPDX-License-Identifier: BUSL-1.1\n"), buf.Bytes()...), nil
}

func catalogSemanticallyEqual(a *catalog, b *catalog) bool {
	if a == nil || b == nil {
		return a == b
	}
	normalizedA := normalizedCatalog(a)
	normalizedB := normalizedCatalog(b)
	return reflect.DeepEqual(normalizedA, normalizedB)
}

func normalizedCatalog(c *catalog) catalog {
	out := *c
	out.Policies = append([]policyEntry(nil), c.Policies...)
	for i := range out.Policies {
		out.Policies[i].MappingRefs = normalizedStringSlice(out.Policies[i].MappingRefs)
		out.Policies[i].KyvernoRules = normalizedStringSlice(out.Policies[i].KyvernoRules)
		out.Policies[i].Upstream.Titles = normalizedStringSlice(out.Policies[i].Upstream.Titles)
		out.Policies[i].Upstream.Categories = normalizedStringSlice(out.Policies[i].Upstream.Categories)
		out.Policies[i].Upstream.PolicyResources = sortedResources(out.Policies[i].Upstream.PolicyResources)
	}
	sort.SliceStable(out.Policies, func(i, j int) bool {
		return out.Policies[i].KyvernoPolicy < out.Policies[j].KyvernoPolicy
	})

	out.Mappings = append([]mappingEntry(nil), c.Mappings...)
	for i := range out.Mappings {
		out.Mappings[i].KyvernoRules = normalizedStringSlice(out.Mappings[i].KyvernoRules)
		out.Mappings[i].MondooChecks = normalizedStringSlice(out.Mappings[i].MondooChecks)
	}
	sort.SliceStable(out.Mappings, func(i, j int) bool {
		return out.Mappings[i].UID < out.Mappings[j].UID
	})
	return out
}

func normalizedStringSlice(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	if out == nil {
		return []string{}
	}
	return out
}

func sortedKeys(items map[string]struct{}) []string {
	out := make([]string, 0, len(items))
	for item := range items {
		if item != "" {
			out = append(out, item)
		}
	}
	sort.Strings(out)
	return out
}

func sortedResources(resources []policyResource) []policyResource {
	out := append([]policyResource(nil), resources...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		if out[i].APIVersion != out[j].APIVersion {
			return out[i].APIVersion < out[j].APIVersion
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}

func annotationValueList(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return []string{value}
}

func stringFromPath(m map[string]any, path ...string) string {
	cur := any(m)
	for _, segment := range path {
		curMap, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = curMap[segment]
	}
	return stringFromAny(cur)
}

func mapFromPath(m map[string]any, path ...string) map[string]any {
	cur := any(m)
	for _, segment := range path {
		curMap, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = curMap[segment]
	}
	return mapFromAny(cur)
}

func mapsFromPath(m map[string]any, path ...string) []map[string]any {
	cur := any(m)
	for _, segment := range path {
		curMap, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = curMap[segment]
	}
	switch items := cur.(type) {
	case []any:
		out := make([]map[string]any, 0, len(items))
		for _, item := range items {
			if itemMap := mapFromAny(item); len(itemMap) > 0 {
				out = append(out, itemMap)
			}
		}
		return out
	case map[string]any:
		if len(items) == 0 {
			return nil
		}
		return []map[string]any{items}
	default:
		return nil
	}
}

func mapFromAny(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

func stringFromMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	return stringFromAny(m[key])
}

func stringFromAny(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprintf("%v", t)
	}
}

func fatalf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
