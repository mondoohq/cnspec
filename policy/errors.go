// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import "github.com/cockroachdb/errors"

var ErrRiskNotFound = errors.New("risk not found")
var ErrResourceNotFound = errors.New("resource not found")
var ErrAssetNotFound = errors.New("asset not found")
