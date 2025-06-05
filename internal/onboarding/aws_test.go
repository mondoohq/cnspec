// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package onboarding_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	subject "go.mondoo.com/cnspec/v11/internal/onboarding"
)

func TestGenerateAwsHCL_KeyBased(t *testing.T) {
	code, err := subject.GenerateAwsHCL(subject.AwsIntegration{
		Name:  "test-key-integration",
		Space: "space-123",
	})
	assert.Nil(t, err)

	expected := `terraform {
  required_providers {
    mondoo = {
      source  = "mondoohq/mondoo"
      version = "~> 0.19"
    }
  }
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

provider "mondoo" {
  space = "space-123"
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

func TestGenerateAwsHCL_RoleBased(t *testing.T) {
	code, err := subject.GenerateAwsHCL(subject.AwsIntegration{
		Name:       "test-role-integration",
		RoleArn:    "arn:aws:iam::123456789012:role/MondooRole",
		ExternalID: "external-123",
	})
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
}

resource "mondoo_integration_aws" "this" {
  credentials = {
    role = {
      external_id = "external-123"
      role_arn    = "arn:aws:iam::123456789012:role/MondooRole"
    }
  }
  name = "test-role-integration"
}
`
	assert.Equal(t, expected, code)
}

func TestGenerateAwsHCL_ErrorsOnNoAuthMethod(t *testing.T) {
	_, err := subject.GenerateAwsHCL(subject.AwsIntegration{
		Name: "test-integration",
	})
	if err == nil {
		t.Fatal("expected error for no auth method selected, got nil")
	}
}
