// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/v13"
	"go.mondoo.com/mql/v13/mqlc"
	"go.mondoo.com/mql/v13/providers-sdk/v1/testutils"
)

func TestMquery_Whitespaces(t *testing.T) {
	coreSchema := testutils.MustLoadSchema(testutils.SchemaProvider{Provider: "core"})
	conf := mqlc.NewConfig(coreSchema, mql.DefaultFeatures)

	mq := &Mquery{
		Mql: "  mondoo { version \n}   \t\n  ",
	}

	mqexpect := &Mquery{
		Mql: "mondoo { version \n}",
	}

	bundle, err := mq.RefreshChecksumAndType(nil, nil, conf)
	assert.NoError(t, err)
	assert.NotNil(t, bundle)

	bundle, err = mqexpect.RefreshChecksumAndType(nil, nil, conf)
	assert.NoError(t, err)
	assert.NotNil(t, bundle)

	assert.Equal(t, mqexpect.CodeId, mq.CodeId)
}

func TestMquery_CodeIDs(t *testing.T) {
	coreSchema := testutils.MustLoadSchema(testutils.SchemaProvider{Provider: "core"})
	conf := mqlc.NewConfig(coreSchema, mql.DefaultFeatures)

	mqAssetFilter := &Mquery{
		Mql: "mondoo { version \n}",
	}

	mqReg := &Mquery{
		Mql: "mondoo { version \n}",
	}

	_, err := mqAssetFilter.RefreshAsFilter("//some.mrn", conf)
	assert.NoError(t, err)

	_, err = mqReg.RefreshChecksumAndType(nil, nil, conf)
	assert.NoError(t, err)

	assert.Equal(t, mqReg.CodeId, mqAssetFilter.CodeId)
}
