// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"go.mondoo.com/cnquery/v9/explorer"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/testutils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMquery_Whitespaces(t *testing.T) {
	coreSchema := testutils.MustLoadSchema(testutils.SchemaProvider{Provider: "core"})

	mq := &explorer.Mquery{
		Mql: "  mondoo { version \n}   \t\n  ",
	}

	mqexpect := &explorer.Mquery{
		Mql: "mondoo { version \n}",
	}

	bundle, err := mq.RefreshChecksumAndType(nil, nil, coreSchema)
	assert.NoError(t, err)
	assert.NotNil(t, bundle)

	bundle, err = mqexpect.RefreshChecksumAndType(nil, nil, coreSchema)
	assert.NoError(t, err)
	assert.NotNil(t, bundle)

	assert.Equal(t, mqexpect.CodeId, mq.CodeId)
}

func TestMquery_CodeIDs(t *testing.T) {
	coreSchema := testutils.MustLoadSchema(testutils.SchemaProvider{Provider: "core"})
	mqAssetFilter := &explorer.Mquery{
		Mql: "mondoo { version \n}",
	}

	mqReg := &explorer.Mquery{
		Mql: "mondoo { version \n}",
	}

	_, err := mqAssetFilter.RefreshAsFilter("//some.mrn", coreSchema)
	assert.NoError(t, err)

	_, err = mqReg.RefreshChecksumAndType(nil, nil, coreSchema)
	assert.NoError(t, err)

	assert.Equal(t, mqReg.CodeId, mqAssetFilter.CodeId)
}
