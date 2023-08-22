// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMquery_Whitespaces(t *testing.T) {
	mq := DeprecatedV7_Mquery{
		Query: "  mondoo { version \n}   \t\n  ",
	}

	mqexpect := DeprecatedV7_Mquery{
		Query: "mondoo { version \n}",
	}

	bundle, err := mq.RefreshChecksumAndType(nil)
	assert.NoError(t, err)
	assert.NotNil(t, bundle)

	bundle, err = mqexpect.RefreshChecksumAndType(nil)
	assert.NoError(t, err)
	assert.NotNil(t, bundle)

	assert.Equal(t, mqexpect.CodeId, mq.CodeId)
}

func TestMquery_CodeIDs(t *testing.T) {
	mqAssetFilter := DeprecatedV7_Mquery{
		Query: "mondoo { version \n}",
	}

	mqReg := DeprecatedV7_Mquery{
		Query: "mondoo { version \n}",
	}

	_, err := mqAssetFilter.RefreshAsAssetFilter("//some.mrn")
	assert.NoError(t, err)

	_, err = mqReg.RefreshChecksumAndType(nil)
	assert.NoError(t, err)

	assert.Equal(t, mqReg.CodeId, mqAssetFilter.CodeId)
}
