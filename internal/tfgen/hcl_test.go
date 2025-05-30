// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package tfgen_test

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.mondoo.com/cnspec/v11/internal/tfgen"
)

func TestRealGcpHCLGeneration(t *testing.T) {
	expectedOutput := `provider "mondoo" {
  space = "hungry-poet-123456"
}

provider "google" {
  project = "prod-project-123"
  region  = "us-central1"
}

resource "google_service_account" "mondoo" {
  account_id   = "mondoo-integration"
  display_name = "Mondoo service account"
}

resource "google_service_account_key" "mondoo" {
  service_account_id = google_service_account.mondoo.name
}

resource "mondoo_integration_gcp" "production" {
  credentials = {
    private_key = base64decode(google_service_account_key.mondoo.private_key)
  }
  name       = "Production account"
  project_id = "prod-project-123"
}
`
	mondooProvider, err := tfgen.NewProvider("mondoo", tfgen.HclProviderWithAttributes(
		map[string]interface{}{
			"space": "hungry-poet-123456",
		},
	)).ToBlock()
	assert.NoError(t, err)
	googleProvider, err := tfgen.NewProvider("google", tfgen.HclProviderWithAttributes(
		map[string]interface{}{
			"project": "prod-project-123",
			"region":  "us-central1",
		},
	)).ToBlock()
	assert.NoError(t, err)
	googleServiceAccountResource, err := tfgen.NewResource("google_service_account",
		"mondoo", tfgen.HclResourceWithAttributesAndProviderDetails(
			map[string]interface{}{
				"account_id":   "mondoo-integration",
				"display_name": "Mondoo service account",
			}, nil,
		)).ToBlock()
	assert.NoError(t, err)
	googleServiceAccountKey, err := tfgen.NewResource("google_service_account_key",
		"mondoo", tfgen.HclResourceWithAttributesAndProviderDetails(
			map[string]interface{}{
				"service_account_id": tfgen.CreateSimpleTraversal("google_service_account", "mondoo", "name"),
			}, nil,
		)).ToBlock()
	assert.NoError(t, err)
	mondooIntegrationGCP, err := tfgen.NewResource("mondoo_integration_gcp",
		"production", tfgen.HclResourceWithAttributesAndProviderDetails(
			map[string]interface{}{
				"name":       "Production account",
				"project_id": "prod-project-123",
				"credentials": map[string]interface{}{
					"private_key": tfgen.NewFuncCall(
						"base64decode", tfgen.CreateSimpleTraversal("google_service_account_key", "mondoo", "private_key")),
				},
			}, nil,
		)).ToBlock()
	assert.NoError(t, err)

	blocksOutput := tfgen.CreateHclStringOutput(
		tfgen.CombineHclBlocks(
			mondooProvider,
			googleProvider,
			googleServiceAccountResource,
			googleServiceAccountKey,
			mondooIntegrationGCP,
		)...,
	)
	assert.Equal(t, expectedOutput, blocksOutput)

}

func TestProviderToBlock(t *testing.T) {
	provider := tfgen.NewProvider("aws", tfgen.HclProviderWithAttributes(map[string]interface{}{
		"region": "us-west-2",
	}))

	block, err := provider.ToBlock()
	assert.NoError(t, err)
	assert.NotNil(t, block)
	assert.Equal(t, "provider", string(block.Type()))
	expectedOutput := `provider "aws" {
  region = "us-west-2"
}
`
	assert.Equal(t, expectedOutput, tfgen.CreateHclStringOutput(block))
}

func TestProviderWithGenericBlocks(t *testing.T) {
	subBlock := hclwrite.NewBlock("sub_block", []string{})
	provider := tfgen.NewProvider("aws", tfgen.HclProviderWithGenericBlocks(subBlock))

	block, err := provider.ToBlock()
	assert.NoError(t, err)
	assert.NotNil(t, block)
	assert.Len(t, block.Body().Blocks(), 1)
	expectedOutput := `provider "aws" {

  sub_block {
  }
}
`
	assert.Equal(t, expectedOutput, tfgen.CreateHclStringOutput(block))
}

func TestNewRequiredProvider(t *testing.T) {
	provider := tfgen.NewRequiredProvider("aws",
		tfgen.HclRequiredProviderWithSource("hashicorp/aws"),
		tfgen.HclRequiredProviderWithVersion("3.27.0"),
	)

	assert.Equal(t, "hashicorp/aws", provider.Source())
	assert.Equal(t, "3.27.0", provider.Version())
	assert.Equal(t, "aws", provider.Name())
}

func TestCreateRequiredProviders(t *testing.T) {
	provider := tfgen.NewRequiredProvider("mondoo", tfgen.HclRequiredProviderWithSource("mondoohq/mondoo"))
	block, err := tfgen.CreateRequiredProviders(provider)

	assert.NoError(t, err)
	assert.NotNil(t, block)
	assert.Equal(t, "terraform", string(block.Type()))
	expectedOutput := `terraform {
  required_providers {
    mondoo = {
      source = "mondoohq/mondoo"
    }
  }
}
`
	assert.Equal(t, expectedOutput, tfgen.CreateHclStringOutput(block))
}

func TestNewOutput(t *testing.T) {
	output := tfgen.NewOutput("test_output",
		[]string{"aws_instance", "example", "id"},
		"Example description",
	)
	expectedOutput := `output "test_output" {
  description = "Example description"
  value       = aws_instance.example.id
}
`

	block, err := output.ToBlock()
	assert.NoError(t, err)
	assert.NotNil(t, block)
	assert.Equal(t, "output", string(block.Type()))
	assert.Equal(t, expectedOutput, tfgen.CreateHclStringOutput(block))
}
func TestHclResourceToBlock(t *testing.T) {
	resource := tfgen.NewResource("aws_instance",
		"example", tfgen.HclResourceWithAttributesAndProviderDetails(
			map[string]interface{}{"ami": "ami-123456"},
			[]string{"aws.foo"},
		))
	expectedOutput := `resource "aws_instance" "example" {
  ami = "ami-123456"

  provider = aws.foo
}
`
	block, err := resource.ToBlock()
	assert.NoError(t, err)
	assert.NotNil(t, block)
	assert.Equal(t, "resource", string(block.Type()))
	assert.Equal(t, expectedOutput, tfgen.CreateHclStringOutput(block))
}

func TestCombineHclBlocks(t *testing.T) {
	block1 := hclwrite.NewBlock("block1", nil)
	block2 := hclwrite.NewBlock("block2", nil)

	combined := tfgen.CombineHclBlocks(block1, block2)
	assert.Len(t, combined, 2)
}

func TestCreateSimpleTraversal(t *testing.T) {
	traversal := tfgen.CreateSimpleTraversal("aws_instance", "example", "id")
	assert.Len(t, traversal, 3)
	assert.Equal(t, "aws_instance", traversal.RootName())
}

func TestTraversalToString(t *testing.T) {
	traversal := tfgen.CreateSimpleTraversal("aws_instance", "example", "id")
	assert.Equal(t, "aws_instance.example.id", tfgen.TraversalToString(traversal))
}

func TestGenericBlockCreation(t *testing.T) {
	t.Run("should be a working generic block", func(t *testing.T) {
		data, err := tfgen.HclCreateGenericBlock(
			"thing",
			[]string{"a", "b"},
			map[string]interface{}{
				"a": "foo",
				"b": 1,
				"c": false,
				"d": map[string]interface{}{ // Order of map elements should be sorted when executed
					"f": 1,
					"g": "bar",
					"e": true,
				},
				"h": hcl.Traversal{
					hcl.TraverseRoot{
						Name: "module",
					},
					hcl.TraverseAttr{
						Name: "example",
					},
					hcl.TraverseAttr{
						Name: "value",
					},
				},
				"i": []string{"one", "two", "three"},
				"j": []interface{}{"one", 2, true},
				"k": []interface{}{
					map[string]interface{}{"test1": []string{"f", "o", "o"}},
					map[string]interface{}{"test2": []string{"b", "a", "r"}},
				},
			},
		)

		assert.Nil(t, err)
		assert.Equal(t, "thing", data.Type())
		assert.Equal(t, "a", data.Labels()[0])
		assert.Equal(t, "b", data.Labels()[1])
		expectedOutput := `thing "a" "b" {
  a = "foo"
  b = 1
  c = false
  d = {
    e = true
    f = 1
    g = "bar"
  }
  h = module.example.value
  i = ["one", "two", "three"]
  j = ["one", 2, true]
  k = [{
    test1 = ["f", "o", "o"]
    }, {
    test2 = ["b", "a", "r"]
  }]
}
`
		assert.Equal(t, expectedOutput, tfgen.CreateHclStringOutput(data))

	})
	t.Run("should fail to construct generic block with mismatched list element types", func(t *testing.T) {
		_, err := tfgen.HclCreateGenericBlock(
			"thing",
			[]string{},
			map[string]interface{}{
				"k": []map[string]interface{}{ // can use []interface{} here to support this sort of structure, but as-is will fail
					{"test1": []string{"f", "o", "o"}},
					{"test2": []string{"b", "a", "r"}},
				},
			},
		)

		assert.Error(t, err, "should fail to generate block with mismatched list element types")
	})
}

func TestLocalVariable(t *testing.T) {
	local := tfgen.NewLocal("foo", "bar")
	assert.Equal(t, "local.foo", tfgen.TraversalToString(local.TraverseRef()))
	localBlock, err := local.ToBlock()
	require.Nil(t, err)
	assert.Equal(t, `locals {
  foo = "bar"
}
`, tfgen.CreateHclStringOutput(localBlock))
}

func TestModuleBlock(t *testing.T) {
	data, err := tfgen.NewModule("foo",
		"mycorp/mycloud",
		tfgen.HclModuleWithVersion("~> 0.1"),
		tfgen.HclModuleWithAttributes(map[string]interface{}{"bar": "foo"})).ToBlock()

	assert.Nil(t, err)
	assert.Equal(t, "module", data.Type())
	assert.Equal(t, "foo", data.Labels()[0])
	assert.Equal(t,
		"version=\"~> 0.1\"\n",
		string(data.Body().GetAttribute("version").BuildTokens(nil).Bytes()),
	)
	assert.Equal(t,
		"bar=\"foo\"\n",
		string(data.Body().GetAttribute("bar").BuildTokens(nil).Bytes()),
	)
}

func TestModuleWithProviderBlock(t *testing.T) {
	providerDetails := map[string]string{
		"foo.src": "test.abc",
		"foo.dst": "abc.test",
	}

	data, err := tfgen.NewModule("foo",
		"mycorp/mycloud",
		tfgen.HclModuleWithProviderDetails(providerDetails)).ToBlock()

	assert.Nil(t, err)
	assert.Equal(t, "module", data.Type())
	assert.Equal(t, "foo", data.Labels()[0])
	assert.Equal(t,
		"providers= {\nfoo.dst=  abc.test\nfoo.src=  test.abc\n}\n",
		string(data.Body().GetAttribute("providers").BuildTokens(nil).Bytes()))
}

func TestProviderBlock(t *testing.T) {
	attrs := map[string]interface{}{"key": "value"}
	data, err := tfgen.NewProvider("foo", tfgen.HclProviderWithAttributes(attrs)).ToBlock()

	assert.Nil(t, err)
	assert.Equal(t, "provider", data.Type())
	assert.Equal(t, "foo", data.Labels()[0])
	assert.Equal(t, "key=\"value\"\n", string(data.Body().GetAttribute("key").BuildTokens(nil).Bytes()))
}

func TestProviderBlockWithTraversal(t *testing.T) {
	attrs := map[string]interface{}{
		"test": hcl.Traversal{
			hcl.TraverseRoot{Name: "key"},
			hcl.TraverseAttr{Name: "value"},
		}}
	data, err := tfgen.NewProvider("foo", tfgen.HclProviderWithAttributes(attrs)).ToBlock()

	assert.Nil(t, err)
	assert.Equal(t, "provider", data.Type())
	assert.Equal(t, "foo", data.Labels()[0])
	assert.Equal(t, "test=key.value\n", string(data.Body().GetAttribute("test").BuildTokens(nil).Bytes()))
}

func TestRequiredProvidersBlock(t *testing.T) {
	provider1 := tfgen.NewRequiredProvider("foo",
		tfgen.HclRequiredProviderWithSource("test/test"))
	provider2 := tfgen.NewRequiredProvider("bar",
		tfgen.HclRequiredProviderWithVersion("~> 0.1"))
	provider3 := tfgen.NewRequiredProvider("mondoo",
		tfgen.HclRequiredProviderWithSource("mondoohq/mondoo"),
		tfgen.HclRequiredProviderWithVersion("~> 0.19"))
	data, err := tfgen.CreateRequiredProviders(provider1, provider2, provider3)
	assert.Nil(t, err)

	expectedOutput := `terraform {
  required_providers {
    bar = {
      version = "~> 0.1"
    }
    foo = {
      source = "test/test"
    }
    mondoo = {
      source  = "mondoohq/mondoo"
      version = "~> 0.19"
    }
  }
}
`

	assert.Equal(t, expectedOutput, tfgen.CreateHclStringOutput(data))
}

func TestRequiredProvidersBlockWithCustomBlocks(t *testing.T) {
	provider1 := tfgen.NewRequiredProvider("foo",
		tfgen.HclRequiredProviderWithSource("test/test"))
	provider2 := tfgen.NewRequiredProvider("bar",
		tfgen.HclRequiredProviderWithVersion("~> 0.1"))
	provider3 := tfgen.NewRequiredProvider("mondoo",
		tfgen.HclRequiredProviderWithSource("mondoohq/mondoo"),
		tfgen.HclRequiredProviderWithVersion("~> 0.19"))

	customBlock, err := tfgen.HclCreateGenericBlock("backend", []string{"s3"}, nil)
	assert.NoError(t, err)
	data, err := tfgen.CreateRequiredProvidersWithCustomBlocks([]*hclwrite.Block{customBlock}, provider1, provider2, provider3)
	assert.Nil(t, err)

	expectedOutput := `terraform {
  required_providers {
    bar = {
      version = "~> 0.1"
    }
    foo = {
      source = "test/test"
    }
    mondoo = {
      source  = "mondoohq/mondoo"
      version = "~> 0.19"
    }
  }
  backend "s3" {
  }
}
`
	assert.Equal(t, expectedOutput, tfgen.CreateHclStringOutput(data))
}

func TestOutputBlockCreation(t *testing.T) {
	t.Run("should generate correct block for simple output with no description", func(t *testing.T) {
		o := tfgen.NewOutput("test", []string{"test", "one", "two"}, "")
		b, err := o.ToBlock()
		assert.NoError(t, err)
		str := tfgen.CreateHclStringOutput(b)
		assert.Equal(t, "output \"test\" {\n  value = test.one.two\n}\n", str)
	})
	t.Run("should generate correct block for simple output with description", func(t *testing.T) {
		o := tfgen.NewOutput("test", []string{"test", "one", "two"}, "test description")
		b, err := o.ToBlock()
		assert.NoError(t, err)
		str := tfgen.CreateHclStringOutput(b)
		assert.Equal(t, "output \"test\" {\n  description = \"test description\"\n  value       = test.one.two\n}\n", str)
	})
}

func TestModuleToBlock(t *testing.T) {
	module := tfgen.NewModule("vpc", "terraform-aws-modules/vpc/aws",
		tfgen.HclModuleWithAttributes(map[string]interface{}{
			"version": "2.32.0",
		}),
	)

	block, err := module.ToBlock()
	assert.NoError(t, err)
	assert.NotNil(t, block)
	assert.Equal(t, "module", string(block.Type()))
	expectedOutput := `module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "2.32.0"
}
`
	assert.Equal(t, expectedOutput, tfgen.CreateHclStringOutput(block))
}

func TestModuleWithForEach(t *testing.T) {
	forEachValue := map[string]string{
		"dev":  "us-west-1",
		"prod": "us-west-2",
	}

	module := tfgen.NewModule("vpc",
		"terraform-aws-modules/vpc/aws",
		tfgen.HclModuleWithForEach("env", forEachValue),
	)

	block, err := module.ToBlock()
	assert.NoError(t, err)
	assert.NotNil(t, block)
	assert.Len(t, block.Body().Attributes(), 3)
	expectedOutput := `module "vpc" {
  source = "terraform-aws-modules/vpc/aws"

  for_each = {
    dev  = "us-west-1"
    prod = "us-west-2"
  }
  env = each.key
}
`
	assert.Equal(t, expectedOutput, tfgen.CreateHclStringOutput(block))
}
