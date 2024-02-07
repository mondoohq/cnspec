// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"regexp"
	"strings"

	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

// rpcStatus tries to parse an error as a status.Status. If it fails, return a generic
// This can help when we get an rpc error mangled through GraphQL
func rpcStatus(err error) status.Status {
	rpcCode := codes.Unknown
	msg := err.Error()
	wrappedRPCError := regexp.MustCompile("^rpc error: code = ([a-zA-Z]+) desc = (.+)$")
	snakeCase := regexp.MustCompile("([A-Z])")
	m := wrappedRPCError.FindStringSubmatch(err.Error())
	if len(m) == 3 {
		// convert the error code to snake case
		snakeCode := snakeCase.ReplaceAllString(m[1], "_$1")
		snakeCode = strings.TrimPrefix(snakeCode, "_")
		snakeCode = strings.ToUpper(snakeCode)
		stringCode := "\"" + snakeCode + "\""
		err = rpcCode.UnmarshalJSON([]byte(stringCode))
		if err != nil {
			return *status.New(rpcCode, msg)
		}
		msg = m[2]
	}

	return *status.New(rpcCode, msg)
}
