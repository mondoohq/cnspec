// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package onboarding_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	subject "go.mondoo.com/cnspec/v12/internal/onboarding"
)

func TestNewAwsIntegration(t *testing.T) {
	t.Run("all fields are set correctly", func(t *testing.T) {
		actual := subject.NewAwsIntegration("custom-name", "space-123", "access-key", "secret-key")
		expected := subject.NewAwsIntegration("custom-name", "space-123", "access-key", "secret-key")
		assert.Equal(t, expected, actual)
	})

	t.Run("random name is used when no name is provided", func(t *testing.T) {
		awsIntegration := subject.NewAwsIntegration("", "space-123", "access-key", "secret-key")
		assert.Contains(t, awsIntegration.Name, "AWS Integration")
		assert.Len(t, awsIntegration.Name, 25) // "AWS Integration (" + 7 chars + ")"
		assert.Equal(t, "space-123", awsIntegration.Space)
		assert.Equal(t, "access-key", awsIntegration.AccessKey)
		assert.Equal(t, "secret-key", awsIntegration.SecretKey)
	})
}

func TestAwsIntegration_Validate(t *testing.T) {
	t.Run("valid integration", func(t *testing.T) {
		integration := subject.NewAwsIntegration("valid-integration", "space-123", "access-key", "secret-key")
		errs := integration.Validate()
		assert.Empty(t, errs)
	})

	t.Run("missing access key and secret key", func(t *testing.T) {
		integration := subject.NewAwsIntegration("invalid-integration", "space-123", "", "")
		errs := integration.Validate()
		assert.Len(t, errs, 2)
		assert.Equal(t, "missing AWS access key", errs[0].Error())
		assert.Equal(t, "missing AWS secret key", errs[1].Error())
	})
}

func TestGenerateAwsHCL_KeyBased(t *testing.T) {
	code, err := subject.GenerateAwsHCL(subject.NewAwsIntegration(
		"test-key-integration",
		"space-123",
		"AKIAIOSFODNN7EXAMPLE",
		"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	))
	assert.Nil(t, err)

	expected := `terraform {
  required_providers {
    mondoo = {
      source  = "mondoohq/mondoo"
      version = "~> 0.19"
    }
  }
}

provider "mondoo" {
  space = "space-123"
}

variable "aws_access_key" {
  type        = string
  description = "AWS access key used for authentication"
  sensitive   = true
}

variable "aws_secret_key" {
  type        = string
  description = "AWS secret key used for authentication"
  sensitive   = true
}

resource "mondoo_integration_aws" "this" {
  credentials = {
    key = {
      access_key = var.aws_access_key
      secret_key = var.aws_secret_key
    }
  }
  name = "test-key-integration"
}
`
	assert.Equal(t, expected, code)
}

func TestGenerateAwsHCL_ErrorsOnNoAuthMethod(t *testing.T) {
	_, err := subject.GenerateAwsHCL(subject.NewAwsIntegration(
		"test-integration",
		"space-123",
		"",
		"",
	))
	assert.ErrorContains(t, err, "missing AWS access key")
	assert.ErrorContains(t, err, "missing AWS secret key")
}
