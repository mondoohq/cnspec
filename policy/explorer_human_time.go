// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"encoding/json"
	"errors"

	"go.mondoo.com/mql/v13/utils/timex"
)

func (t *HumanTime) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	// parse as a number
	if data[0] != '"' {
		return json.Unmarshal(data, &t.Seconds)
	}

	// parse as a string
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return errors.New("failed to parse " + string(data) + " as a time string: " + err.Error())
	}

	v, err := timex.Parse(s, "")
	if err != nil {
		return errors.New("failed to parse " + s + " as time: " + err.Error())
	}

	t.Seconds = v.Unix()
	return nil
}
