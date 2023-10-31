// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"github.com/pkg/errors"
	"go.mondoo.com/cnquery/v9/checksums"
	"go.mondoo.com/cnquery/v9/explorer"
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/mrn"
	"sort"
)

func RefreshMRN(ownerMRN string, existingMRN string, resource string, uid string) (string, error) {
	// NOTE: asset policy bundles may not have an owner set, therefore we skip if the query already has an mrn
	if existingMRN != "" {
		if !mrn.IsValid(existingMRN) {
			return "", errors.New("invalid MRN: " + existingMRN)
		}
		return existingMRN, nil
	}

	if ownerMRN == "" {
		return "", errors.New("cannot refresh MRN if the owner MRN is empty")
	}

	if uid == "" {
		return "", errors.New("cannot refresh MRN with an empty UID")
	}

	mrn, err := mrn.NewChildMRN(ownerMRN, resource, uid)
	if err != nil {
		return "", err
	}

	return mrn.String(), nil
}

func ChecksumAssetFilters(queries []*explorer.Mquery, schema llx.Schema) (string, error) {
	for i := range queries {
		if _, err := queries[i].RefreshAsFilter("", schema); err != nil {
			return "", errors.New("failed to compile query: " + err.Error())
		}
	}

	sort.Slice(queries, func(i, j int) bool {
		return queries[i].CodeId < queries[j].CodeId
	})

	afc := checksums.New
	for i := range queries {
		afc = afc.Add(queries[i].CodeId)
	}

	return afc.String(), nil
}

// RefreshChecksums of all queries
// Note: This method is used for testing purposes only. If you need it in other
// places please make sure to implement the query lookup.
func (m *Mqueries) RefreshChecksums(schema llx.Schema, props map[string]explorer.PropertyRef) error {
	queries := map[string]*explorer.Mquery{}
	for i := range m.Items {
		if _, err := m.Items[i].RefreshChecksumAndType(queries, props, schema); err != nil {
			return err
		}
	}
	return nil
}
