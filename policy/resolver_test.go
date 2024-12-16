// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/testutils"
	"go.mondoo.com/cnspec/v11/internal/datalakes/inmemory"
	"go.mondoo.com/cnspec/v11/policy"
)

type testAsset struct {
	asset      string
	policies   []string
	frameworks []string
}

func parseBundle(t *testing.T, data string) *policy.Bundle {
	res, err := policy.BundleFromYAML([]byte(data))
	require.NoError(t, err)
	return res
}

func initResolver(t *testing.T, assets []*testAsset, bundles []*policy.Bundle) *policy.LocalServices {
	runtime := testutils.LinuxMock()
	_, srv, err := inmemory.NewServices(runtime, nil)
	require.NoError(t, err)

	for i := range bundles {
		bundle := bundles[i]
		_, err := srv.SetBundle(context.Background(), bundle)
		require.NoError(t, err)
	}

	for i := range assets {
		asset := assets[i]
		_, err := srv.Assign(context.Background(), &policy.PolicyAssignment{
			AssetMrn:      asset.asset,
			PolicyMrns:    asset.policies,
			FrameworkMrns: asset.frameworks,
		})
		require.NoError(t, err)
	}

	return srv
}

func policyMrn(uid string) string {
	return "//test.sth/policies/" + uid
}

func frameworkMrn(uid string) string {
	return "//test.sth/frameworks/" + uid
}

func controlMrn(uid string) string {
	return "//test.sth/controls/" + uid
}

func queryMrn(uid string) string {
	return "//test.sth/queries/" + uid
}

func riskFactorMrn(uid string) string {
	return "//test.sth/risks/" + uid
}

func isFramework(queryId string) bool {
	return strings.Contains(queryId, "/frameworks/")
}

func isControl(queryId string) bool {
	return strings.Contains(queryId, "/controls/")
}

func isPolicy(queryId string) bool {
	return strings.Contains(queryId, "/policies/")
}
