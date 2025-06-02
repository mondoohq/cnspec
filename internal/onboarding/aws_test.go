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
		Name:      "test-key-integration",
		Space:     "space-123",
		AccessKey: "AKIAXXXXXXXXXXXXXXXX",
		SecretKey: "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
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
  space = "space-123"
}

resource "mondoo_integration_aws" "this" {
  credentials = {
    key = {
      access_key = "AKIAXXXXXXXXXXXXXXXX"
      secret_key = "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
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
  space = ""
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

func TestGenerateAwsHCL_Minimal(t *testing.T) {
	code, err := subject.GenerateAwsHCL(subject.AwsIntegration{})
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
  space = ""
}

resource "mondoo_integration_aws" "this" {
  credentials = {
    role = {
      external_id = ""
      role_arn    = ""
    }
  }
  name = "AWS Integration"
}
`
	assert.Equal(t, expected, code)
}
