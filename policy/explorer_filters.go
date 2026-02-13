// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/checksums"
	"go.mondoo.com/mql/v13/mqlc"
	"go.mondoo.com/mql/v13/utils/multierr"
	"go.mondoo.com/mql/v13/utils/sortx"
)

// NewFilters creates a Filters object from a simple list of MQL snippets
func NewFilters(queries ...string) *Filters {
	res := &Filters{
		Items: map[string]*Mquery{},
	}

	for i := range queries {
		res.Items[strconv.Itoa(i)] = &Mquery{Mql: queries[i]}
	}

	return res
}

// Checksum computes the checksum for the filters
func (filters *Filters) Checksum() (checksums.Fast, checksums.Fast) {
	content := checksums.New
	execution := checksums.New

	if filters == nil {
		return content, execution
	}

	keys := make([]string, len(filters.Items))
	i := 0
	for k := range filters.Items {
		if len(k) < 2 {
			panic("internal error processing filter checksums: queries are not compiled")
		}

		keys[i] = k
		i++
	}
	sort.Strings(keys)

	for i := range keys {
		filter := filters.Items[keys[i]]
		content = content.Add(filter.Title).Add(filter.Desc)

		if filter.Checksum == "" || filter.CodeId == "" {
			panic("internal error processing filter checksums: query is compiled")
		}

		content = content.Add(filter.Checksum)
		execution = execution.Add(filter.CodeId)
	}

	content = content.AddUint(uint64(execution))

	return content, execution
}

func (s *Filters) UnmarshalJSON(data []byte) error {
	var str string
	err := json.Unmarshal(data, &str)
	if err == nil {
		s.Items = map[string]*Mquery{}
		s.Items[""] = &Mquery{
			Mql: str,
		}
		return nil
	}

	var list []*Mquery
	err = json.Unmarshal(data, &list)
	if err == nil {
		s.Items = map[string]*Mquery{}
		for i := range list {
			s.Items[strconv.Itoa(i)] = list[i]
		}
		return nil
	}

	type tmp Filters
	return json.Unmarshal(data, (*tmp)(s))
}

func (s *Filters) Compile(ownerMRN string, conf mqlc.CompilerConfig) error {
	if s == nil || len(s.Items) == 0 {
		return nil
	}

	res := make(map[string]*Mquery, len(s.Items))
	for _, query := range s.Items {
		_, err := query.RefreshAsFilter(ownerMRN, conf)
		if err != nil {
			return err
		}

		if _, ok := res[query.CodeId]; ok {
			continue
		}

		res[query.CodeId] = query
	}

	s.Items = res
	return nil
}

func (s *Filters) ComputeChecksum(checksum checksums.Fast, queryMrn string, conf mqlc.CompilerConfig) (checksums.Fast, error) {
	if s == nil {
		return checksum, nil
	}

	keys := sortx.Keys(s.Items)
	for _, k := range keys {
		query := s.Items[k]
		if query.Checksum == "" {
			log.Warn().
				Str("filter", query.Mql).
				Msg("refresh checksum on filters, which should have been pre-compiled")
			_, err := query.RefreshAsFilter(queryMrn, conf)
			if err != nil {
				return checksum, multierr.Wrap(err, "cannot refresh checksum for query, failed to compile")
			}
			if query.Checksum == "" {
				return checksum, errors.New("cannot refresh checksum for query, its filters were not compiled")
			}
		}
		checksum = checksum.Add(query.Checksum)
	}
	return checksum, nil
}

// AddFilters takes all given filters (or nil) and adds them to the parent.
func (s *Filters) AddFilters(child *Filters) {
	if child == nil {
		return
	}

	for k, v := range child.Items {
		s.Items[k] = v
	}
}

var ErrQueryNotFound = errors.New("query not found")

// AddQueryFilters attempts to take a query (or nil) and register all its filters.
func (s *Filters) AddQueryFilters(query *Mquery, lookupQueries map[string]*Mquery) error {
	if query == nil {
		return nil
	}

	return s.AddQueryFiltersFn(context.Background(), query, func(_ context.Context, mrn string) (*Mquery, error) {
		q, ok := lookupQueries[mrn]
		if !ok {
			return nil, ErrQueryNotFound
		}
		return q, nil
	})
}

// AddQueryFiltersFn attempts to take a query (or nil) and register all its filters.
func (s *Filters) AddQueryFiltersFn(ctx context.Context, query *Mquery, lookupQuery func(ctx context.Context, mrn string) (*Mquery, error)) error {
	if query == nil {
		return nil
	}

	s.AddFilters(query.Filters)

	for i := range query.Variants {
		mrn := query.Variants[i].Mrn
		variant, err := lookupQuery(ctx, mrn)
		if err != nil {
			return multierr.Wrap(err, "cannot find query variant "+mrn)
		}
		s.AddQueryFiltersFn(ctx, variant, lookupQuery)
	}
	return nil
}

// Supports checks if the given queries (via CodeIDs) are supported by this set of
// asset filters.
func (s *Filters) Supports(supported map[string]struct{}) bool {
	if s == nil || len(s.Items) == 0 {
		return true
	}

	for k := range s.Items {
		if _, ok := supported[k]; ok {
			return true
		}
	}

	return false
}

func (s *Filters) Summarize() string {
	if s == nil || len(s.Items) == 0 {
		return ""
	}

	filters := make([]string, len(s.Items))
	i := 0
	for _, filter := range s.Items {
		if filter.Title != "" {
			filters[i] = filter.Title
		} else {
			filters[i] = filter.Mql
		}
		i++
	}

	sort.Strings(filters)
	return strings.Join(filters, ", ")
}
