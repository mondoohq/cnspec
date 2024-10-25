// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

// A package that generates Terraform deployment code.
package tfgen

import (
	"errors"
	"fmt"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

type HclProvider struct {
	// Required. Provider name.
	name string

	// Optional. Extra properties for this module.
	// Can supply string, bool, int, or map[string]interface{} as values
	attributes map[string]interface{}

	// Optional. Generic blocks
	blocks []*hclwrite.Block
}

func (p *HclProvider) ToBlock() (*hclwrite.Block, error) {
	block, err := HclCreateGenericBlock("provider", []string{p.name}, p.attributes)
	if err != nil {
		return nil, err
	}

	if p.blocks != nil {
		for _, b := range p.blocks {
			block.Body().AppendNewline()
			block.Body().AppendBlock(b)
		}
	}

	return block, nil
}

type HclProviderModifier func(p *HclProvider)

// NewProvider Create a new HCL Provider
func NewProvider(name string, mods ...HclProviderModifier) *HclProvider {
	provider := &HclProvider{name: name}
	for _, m := range mods {
		m(provider)
	}
	return provider
}

func HclProviderWithAttributes(attrs map[string]interface{}) HclProviderModifier {
	return func(p *HclProvider) {
		p.attributes = attrs
	}
}

// HclProviderWithGenericBlocks sets the generic blocks within the provider.
func HclProviderWithGenericBlocks(blocks ...*hclwrite.Block) HclProviderModifier {
	return func(p *HclProvider) {
		p.blocks = blocks
	}
}

type HclRequiredProvider struct {
	name    string
	source  string
	version string
}

func (p *HclRequiredProvider) Source() string {
	return p.source
}

func (p *HclRequiredProvider) Version() string {
	return p.version
}

func (p *HclRequiredProvider) Name() string {
	return p.name
}

type HclRequiredProviderModifier func(p *HclRequiredProvider)

func HclRequiredProviderWithSource(source string) HclRequiredProviderModifier {
	return func(p *HclRequiredProvider) {
		p.source = source
	}
}

func HclRequiredProviderWithVersion(version string) HclRequiredProviderModifier {
	return func(p *HclRequiredProvider) {
		p.version = version
	}
}

func NewRequiredProvider(name string, mods ...HclRequiredProviderModifier) *HclRequiredProvider {
	provider := &HclRequiredProvider{name: name}
	for _, m := range mods {
		m(provider)
	}
	return provider
}

type ForEach struct {
	key   string
	value map[string]string
}

type HclOutput struct {
	// Required. Name of the resultant output.
	name string

	// Required. Converted into a traversal.
	// e.g. []string{"a", "b", "c"} as input results in traversal having value a.b.c
	value []string

	// Optional.
	description string
}

func (m *HclOutput) ToBlock() (*hclwrite.Block, error) {
	if m.value == nil {
		return nil, errors.New("value must be supplied")
	}

	attributes := map[string]interface{}{
		"value": CreateSimpleTraversal(m.value...),
	}

	if m.description != "" {
		attributes["description"] = m.description
	}

	block, err := HclCreateGenericBlock(
		"output",
		[]string{m.name},
		attributes,
	)
	if err != nil {
		return nil, err
	}

	return block, nil
}

// NewOutput Create a provider statement in the HCL output.
func NewOutput(name string, value []string, description string) *HclOutput {
	return &HclOutput{name: name, description: description, value: value}
}

type HclModule struct {
	// Required. Module name.
	name string

	// Required. Source for this module.
	source string

	// Required. Version.
	version string

	// Optional. Extra properties for this module.
	// Can supply string, bool, int, or map[string]interface{} as values
	attributes map[string]interface{}

	// Optional. Provide a map of strings. Creates an instance of the module block for each item in the map, with the
	// map keys assigned to the key field.
	forEach *ForEach

	// Optional. Provider details to override defaults. These values must be supplied as strings, and raw values will be
	// accepted. Unfortunately map[string]hcl.Traversal is not a format that is supported by hclwrite.SetAttributeValue
	// today so we must work around it (https://github.com/hashicorp/hcl/issues/347).
	providerDetails map[string]string
}

type HclModuleModifier func(p *HclModule)

// NewModule Create a provider statement in the HCL output.
func NewModule(name string, source string, mods ...HclModuleModifier) *HclModule {
	module := &HclModule{name: name, source: source}
	for _, m := range mods {
		m(module)
	}
	return module
}

// HclModuleWithAttributes Used to set parameters within the module usage.
func HclModuleWithAttributes(attrs map[string]interface{}) HclModuleModifier {
	return func(p *HclModule) {
		p.attributes = attrs
	}
}

// HclModuleWithVersion Used to set the version of a module source to use.
func HclModuleWithVersion(version string) HclModuleModifier {
	return func(p *HclModule) {
		p.version = version
	}
}

// HclModuleWithProviderDetails Used to provide additional provider details to a given module.
//
// Note: The values supplied become traversals
//
//	e.g. https://www.terraform.io/docs/language/modules/develop/providers.html#passing-providers-explicitly
func HclModuleWithProviderDetails(providerDetails map[string]string) HclModuleModifier {
	return func(p *HclModule) {
		p.providerDetails = providerDetails
	}
}

func HclModuleWithForEach(key string, value map[string]string) HclModuleModifier {
	return func(p *HclModule) {
		p.forEach = &ForEach{key, value}
	}
}

// ToBlock Create hclwrite.Block for module.
func (m *HclModule) ToBlock() (*hclwrite.Block, error) {
	if m.attributes == nil {
		m.attributes = make(map[string]interface{})
	}
	if m.source != "" {
		m.attributes["source"] = m.source

	}
	if m.version != "" {
		m.attributes["version"] = m.version
	}
	block, err := HclCreateGenericBlock(
		"module",
		[]string{m.name},
		m.attributes,
	)
	if err != nil {
		return nil, err
	}

	if m.forEach != nil {
		block.Body().AppendNewline()

		value, err := convertTypeToCty(m.forEach.value)
		if err != nil {
			return nil, err
		}
		block.Body().SetAttributeValue("for_each", value)

		block.Body().SetAttributeRaw(m.forEach.key, createForEachKey())
	}

	if m.providerDetails != nil {
		block.Body().AppendNewline()
		block.Body().SetAttributeRaw("providers", CreateMapTraversalTokens(m.providerDetails))
	}

	return block, nil
}

// ToResourceBlock Create hclwrite.Block for resource.
func (m *HclResource) ToResourceBlock() (*hclwrite.Block, error) {
	if m.attributes == nil {
		m.attributes = make(map[string]interface{})
	}

	block, err := HclCreateGenericBlock(
		"resource",
		[]string{m.rType, m.name},
		m.attributes,
	)
	if err != nil {
		return nil, err
	}

	if m.providerDetails != nil {
		block.Body().AppendNewline()
		block.Body().SetAttributeTraversal("provider", CreateSimpleTraversal(m.providerDetails...))
	}

	if m.blocks != nil {
		for _, b := range m.blocks {
			block.Body().AppendNewline()
			block.Body().AppendBlock(b)
		}
	}

	return block, nil
}

type HclResource struct {
	// Required. Resource type.
	rType string

	// Required. Resource name.
	name string

	// Optional. Extra properties for this resource.
	// Can supply string, bool, int, or map[string]interface{} as values
	attributes map[string]interface{}

	// Optional. Provider details to override defaults.
	// These values must be supplied as strings, and raw values will be accepted.Unfortunately
	// map[string]hcl.Traversal is not a format that is supported by hclwrite.SetAttributeValue
	// today so we must work around it (https://github.com/hashicorp/hcl/issues/347).
	providerDetails []string

	// Optional. Generic blocks
	blocks []*hclwrite.Block
}

type HclResourceModifier func(p *HclResource)

// NewResource Create a provider statement in the HCL output.
func NewResource(rType string, name string, mods ...HclResourceModifier) *HclResource {
	resource := &HclResource{rType: rType, name: name}
	for _, m := range mods {
		m(resource)
	}
	return resource
}

// HclResourceWithAttributesAndProviderDetails Used to set parameters within the resource usage.
func HclResourceWithAttributesAndProviderDetails(attrs map[string]interface{},
	providerDetails []string) HclResourceModifier {
	return func(p *HclResource) {
		p.attributes = attrs
		p.providerDetails = providerDetails
	}
}

// HclResourceWithGenericBlocks sets the generic blocks within the resource.
func HclResourceWithGenericBlocks(blocks ...*hclwrite.Block) HclResourceModifier {
	return func(p *HclResource) {
		p.blocks = blocks
	}
}

// Convert standard value types to cty.Value.
//
// All values used in hclwrite.Block(s) must be cty.Value or a cty.Traversal.
// This function performs that conversion for standard types (non-traversal).
func convertTypeToCty(value interface{}) (cty.Value, error) {
	switch v := value.(type) {
	case string:
		return cty.StringVal(v), nil
	case int:
		return cty.NumberIntVal(int64(v)), nil
	case int64:
		return cty.NumberIntVal(int64(v)), nil
	case bool:
		return cty.BoolVal(v), nil
	case map[string]string:
		if len(v) == 0 {
			return cty.NilVal, nil
		}
		valueMap := map[string]cty.Value{}
		for key, val := range v {
			valueMap[key] = cty.StringVal(val)
		}
		return cty.MapVal(valueMap), nil
	case map[string]interface{}:
		if len(v) == 0 {
			return cty.NilVal, nil
		}
		valueMap := map[string]cty.Value{}
		for key, val := range v {
			convertedValue, err := convertTypeToCty(val)
			if err != nil {
				return cty.NilVal, err
			}
			valueMap[key] = convertedValue
		}
		return cty.MapVal(valueMap), nil
	case []map[string]interface{}:
		values := []cty.Value{}
		for _, item := range v {
			valueMap := map[string]cty.Value{}
			for key, val := range item {
				convertedValue, err := convertTypeToCty(val)
				if err != nil {
					return cty.NilVal, err
				}
				valueMap[key] = convertedValue
			}
			values = append(values, cty.ObjectVal(valueMap))
		}
		return cty.ListVal(values), nil
	case []string:
		if len(v) == 0 {
			return cty.ListValEmpty(cty.String), nil
		}
		valueSlice := []cty.Value{}
		for _, s := range v {
			valueSlice = append(valueSlice, cty.StringVal(s))
		}
		return cty.ListVal(valueSlice), nil
	case []interface{}:
		valueSlice := []cty.Value{}
		for _, i := range v {
			newVal, err := convertTypeToCty(i)
			if err != nil {
				return cty.Value{}, err
			}
			valueSlice = append(valueSlice, newVal)
		}
		return cty.TupleVal(valueSlice), nil
	default:
		return cty.NilVal, errors.New("unknown attribute value type")
	}
}

// Used to set block attribute values based on attribute value interface type.
//
// hclwrite.Block attributes use cty.Value, hclwrite.Tokens or can be traversals, this function
// determines what type of value is being used and builds the block accordingly.
func setBlockAttributeValue(block *hclwrite.Block, key string, val interface{}) error {
	switch v := val.(type) {
	case hcl.Traversal:
		block.Body().SetAttributeTraversal(key, v)
	case hclwrite.Tokens:
		block.Body().SetAttributeRaw(key, v)
	case string, int, bool, []string, []interface{}, map[string]string:
		value, err := convertTypeToCty(v)
		if err != nil {
			return err
		}
		block.Body().SetAttributeValue(key, value)
	case []map[string]interface{}:
		values := []cty.Value{}
		for _, item := range v {
			valueMap := map[string]cty.Value{}
			for key, val := range item {
				convertedValue, err := convertTypeToCty(val)
				if err != nil {
					return err
				}
				valueMap[key] = convertedValue
			}
			values = append(values, cty.ObjectVal(valueMap))
		}

		if !cty.CanListVal(values) {
			return errors.New(
				"setBlockAttributeValue: Values can not be coalesced into a single List due to inconsistent element types",
			)
		}
		block.Body().SetAttributeValue(key, cty.ListVal(values))
	case map[string]interface{}:
		var keys []string
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		objects := []hclwrite.ObjectAttrTokens{}
		for _, attrKey := range keys {
			attrVal := v[attrKey]
			t, ok := attrVal.(hclwrite.Tokens)
			if !ok {
				value, err := convertTypeToCty(attrVal)
				if err != nil {
					return err
				}
				t = hclwrite.TokensForValue(value)
			}
			objects = append(objects, hclwrite.ObjectAttrTokens{
				Name:  hclwrite.TokensForIdentifier(attrKey),
				Value: t,
			})
		}

		block.Body().SetAttributeRaw(key, hclwrite.TokensForObject(objects))
	default:
		return fmt.Errorf("setBlockAttributeValue: unknown type for key: %s", key)
	}

	return nil
}

// HclCreateGenericBlock Helper to create various types of new hclwrite.Block using generic inputs.
func HclCreateGenericBlock(hcltype string, labels []string, attr map[string]interface{}) (*hclwrite.Block, error) {
	block := hclwrite.NewBlock(hcltype, labels)

	// Source and version require some special handling, should go at the top of a block declaration
	sourceFound := false
	versionFound := false

	// We need/want to guarantee the ordering of the attributes, do that here
	var keys []string
	for k := range attr {
		switch k {
		case "source":
			sourceFound = true
		case "version":
			versionFound = true
		default:
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	if sourceFound || versionFound {
		var newKeys []string
		if sourceFound {
			newKeys = append(newKeys, "source")
		}
		if versionFound {
			newKeys = append(newKeys, "version")
		}
		keys = append(newKeys, keys...)
	}

	// Write block data
	for _, key := range keys {
		val := attr[key]
		if err := setBlockAttributeValue(block, key, val); err != nil {
			return nil, err
		}
	}

	return block, nil
}

// Create tokens for map of traversals. Used as a workaround for writing complex types where
// the built-in SetAttributeValue won't work.
func CreateMapTraversalTokens(input map[string]string) hclwrite.Tokens {
	// Sort input
	var keys []string
	for k := range input {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenOBrace, Bytes: []byte("{"), SpacesBefore: 1},
		{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
	}

	for _, k := range keys {
		tokens = append(tokens, []*hclwrite.Token{
			{Type: hclsyntax.TokenStringLit, Bytes: []byte(k)},
			{Type: hclsyntax.TokenEqual, Bytes: []byte("=")},
			{Type: hclsyntax.TokenStringLit, Bytes: []byte(" " + input[k]), SpacesBefore: 1},
			{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
		}...)
	}

	tokens = append(tokens, []*hclwrite.Token{
		{Type: hclsyntax.TokenNewline},
		{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")},
	}...)

	return tokens
}

// Create tokens for the for_each meta-argument.
func createForEachKey() hclwrite.Tokens {
	return hclwrite.Tokens{
		{Type: hclsyntax.TokenStringLit, Bytes: []byte(" each.key"), SpacesBefore: 1},
	}
}

// CreateHclStringOutput Convert blocks to a string.
func CreateHclStringOutput(blocks ...*hclwrite.Block) string {
	file := hclwrite.NewEmptyFile()
	body := file.Body()
	blockCount := len(blocks) - 1

	for i, b := range blocks {
		if b != nil {
			body.AppendBlock(b)

			// If this is not the last block, add a new line to provide spacing
			if i < blockCount {
				body.AppendNewline()
			}
		}
	}
	return string(file.Bytes())
}

// rootTerraformBlock is a helper that creates the literal `terraform{}` hcl block.
func rootTerraformBlock() (*hclwrite.Block, error) {
	return HclCreateGenericBlock("terraform", nil, nil)
}

// createRequiredProviders is a helper that creates the `required_providers` hcl block.
func createRequiredProviders(providers ...*HclRequiredProvider) (*hclwrite.Block, error) {
	providerDetails := map[string]interface{}{}
	for _, provider := range providers {
		details := map[string]interface{}{}
		if provider.Source() != "" {
			details["source"] = provider.Source()
		}
		if provider.Version() != "" {
			details["version"] = provider.Version()
		}
		providerDetails[provider.Name()] = details
	}

	requiredProviders, err := HclCreateGenericBlock("required_providers", nil, providerDetails)
	if err != nil {
		return nil, err
	}

	return requiredProviders, nil
}

// CreateRequiredProviders Create required providers block.
func CreateRequiredProviders(providers ...*HclRequiredProvider) (*hclwrite.Block, error) {
	block, err := rootTerraformBlock()
	if err != nil {
		return nil, err
	}

	requiredProviders, err := createRequiredProviders(providers...)
	if err != nil {
		return nil, err
	}

	block.Body().AppendBlock(requiredProviders)
	return block, nil
}

// CreateRequiredProviders Create required providers block.
func CreateRequiredProvidersWithCustomBlocks(
	blocks []*hclwrite.Block,
	providers ...*HclRequiredProvider,
) (*hclwrite.Block, error) {
	block, err := rootTerraformBlock()
	if err != nil {
		return nil, err
	}

	requiredProviders, err := createRequiredProviders(providers...)
	if err != nil {
		return nil, err
	}

	block.Body().AppendBlock(requiredProviders)
	for _, customBlock := range blocks {
		block.Body().AppendBlock(customBlock)
	}

	return block, nil
}

// CombineHclBlocks Simple helper to combine multiple blocks (or slices of blocks) into a
// single slice to be rendered to string.
func CombineHclBlocks(results ...interface{}) []*hclwrite.Block {
	blocks := []*hclwrite.Block{}
	// Combine all blocks into single flat slice
	for _, result := range results {
		switch v := result.(type) {
		case *hclwrite.Block:
			if v != nil {
				blocks = append(blocks, v)
			}
		case []*hclwrite.Block:
			if len(v) > 0 {
				blocks = append(blocks, v...)
			}
		default:
			continue
		}
	}

	return blocks
}

// CreateSimpleTraversal helper to create a hcl.Traversal in the order of supplied []string.
//
// e.g. []string{"a", "b", "c"} as input results in traversal having value a.b.c
func CreateSimpleTraversal(input ...string) hcl.Traversal {
	var traverser []hcl.Traverser

	for i, val := range input {
		if i == 0 {
			traverser = append(traverser, hcl.TraverseRoot{Name: val})
		} else {
			traverser = append(traverser, hcl.TraverseAttr{Name: val})
		}
	}
	return traverser
}

// NewFuncCall wraps the function name around the traversal and returns hcl tokens
func NewFuncCall(funcName string, traversal hcl.Traversal) hclwrite.Tokens {
	return hclwrite.TokensForFunctionCall(funcName, hclwrite.TokensForTraversal(traversal))
}
