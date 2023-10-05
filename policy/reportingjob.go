// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"sort"

	"go.mondoo.com/cnquery/v9/checksums"
	"go.mondoo.com/cnquery/v9/utils/sortx"
)

// RefreshChecksum recalculates the reporting job checksum
func (r *ReportingJob) RefreshChecksum() {
	checksum := checksums.New
	checksum = checksum.Add("v2")
	checksum = checksum.Add(r.Uuid)
	checksum = checksum.Add(r.QrId)

	{
		jobIDs := sortx.Keys(r.ChildJobs)
		for i := range jobIDs {
			key := jobIDs[i]
			impact := r.ChildJobs[key]
			checksum = checksum.Add(key)
			if impact != nil {
				checksum = checksum.AddUint(impact.Checksum())
			}
		}
	}

	{
		notify := make([]string, len(r.Notify))
		copy(notify, r.Notify)
		sort.Strings(notify)
		for i := range notify {
			checksum = checksum.Add(notify[i])
		}
	}
	r.Checksum = checksum.String()
}
