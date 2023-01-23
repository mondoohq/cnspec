package policy

import (
	"sort"

	"go.mondoo.com/cnquery/checksums"
)

// RefreshChecksum recalculates the reporting job checksum
func (r *ReportingJob) RefreshChecksum() {
	checksum := checksums.New
	checksum = checksum.Add("v2")
	checksum = checksum.Add(r.Uuid)
	checksum = checksum.Add(r.QrId)

	{
		specKeys := make([]string, len(r.Spec))
		i := 0
		for k := range r.Spec {
			specKeys[i] = k
			i++
		}
		sort.Strings(specKeys)
		for i := range specKeys {
			key := specKeys[i]
			impact := r.Spec[key]
			checksum = checksum.Add(key)
			if impact != nil {
				checksum = checksum.
					AddUint(uint64(impact.Scoring)).
					AddUint(uint64(impact.Value)).
					AddUint(uint64(impact.Weight))
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
