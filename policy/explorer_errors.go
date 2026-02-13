// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

type ErrorStatusCode uint32

const (
	Unknown ErrorStatusCode = iota + 100
	NotApplicable
	NoQueries
)

func (e ErrorStatusCode) String() string {
	switch e {
	case NotApplicable:
		return "NotApplicable"
	case NoQueries:
		return "NoQueries"
	default:
		return "Unknown"
	}
}

func NewAssetMatchError(mrn string, objectType string, errorReason string, assetFilters []*Mquery, supported *Filters) error {
	if len(assetFilters) == 0 {
		msg := "asset doesn't support any " + objectType
		st := status.New(codes.InvalidArgument, msg)

		std, err := st.WithDetails(&errdetails.ErrorInfo{
			Domain: POLICY_SERVICE_NAME,
			Reason: errorReason,
			Metadata: map[string]string{
				"mrn":       mrn,
				"errorCode": NotApplicable.String(),
			},
		})
		if err != nil {
			log.Error().Err(err).Msg("could not send status with additional information")
			return st.Err()
		}
		return std.Err()
	}

	supportedSummary := supported.Summarize()
	var supportedPrefix string
	if supportedSummary == "" {
		supportedPrefix = objectType + " didn't provide any filters"
	} else {
		supportedPrefix = objectType + " support: "
	}

	filters := make([]string, len(assetFilters))
	for i := range assetFilters {
		filters[i] = strings.TrimSpace(assetFilters[i].Mql)
	}
	sort.Strings(filters)
	foundSummary := strings.Join(filters, ", ")
	foundPrefix := "asset supports: "

	msg := "asset isn't supported by any " + objectType + "\n" +
		supportedPrefix + supportedSummary + "\n" +
		foundPrefix + foundSummary + "\n"
	return status.Error(codes.InvalidArgument, msg)
}
