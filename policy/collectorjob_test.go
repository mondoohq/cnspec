// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

type zerologMsg struct {
	Error *InvalidCollectorErrorJson `json:"error"`
}

type InvalidCollectorErrorJson struct {
	SpecsByJob     map[string][]string `json:"specs-by-job"`
	NotifiersByJob map[string][]string `json:"notifiers-by-job"`
}

func TestInvalidCollectorJobError(t *testing.T) {
	t.Run("only notifiers", func(t *testing.T) {
		buf := &strings.Builder{}
		log := zerolog.New(buf)

		invalidCollectorError := newInvalidCollectorJobError()
		invalidCollectorError.addInvalidNotifier("a", "b")
		invalidCollectorError.addInvalidNotifier("c", "d")
		invalidCollectorError.addInvalidNotifier("c", "e")

		log.Error().Err(invalidCollectorError).Msg("failed")

		received := InvalidCollectorErrorJson{}
		err := json.Unmarshal([]byte(buf.String()), &zerologMsg{Error: &received})
		require.NoError(t, err)

		require.Equal(t, InvalidCollectorErrorJson{
			NotifiersByJob: map[string][]string{
				"a": {"b"},
				"c": {"d", "e"},
			},
			SpecsByJob: make(map[string][]string),
		}, received, buf.String())
	})

	t.Run("only specs", func(t *testing.T) {
		buf := &strings.Builder{}
		log := zerolog.New(buf)

		invalidCollectorError := newInvalidCollectorJobError()

		invalidCollectorError.addInvalidSpec("f", "g")
		invalidCollectorError.addInvalidSpec("f", "h")
		invalidCollectorError.addInvalidSpec("i", "j")

		log.Error().Err(invalidCollectorError).Msg("failed")

		received := InvalidCollectorErrorJson{}
		err := json.Unmarshal([]byte(buf.String()), &zerologMsg{Error: &received})
		require.NoError(t, err)

		require.Equal(t, InvalidCollectorErrorJson{
			SpecsByJob: map[string][]string{
				"f": {"g", "h"},
				"i": {"j"},
			},
			NotifiersByJob: make(map[string][]string),
		}, received, buf.String())
	})

	t.Run("both", func(t *testing.T) {
		buf := &strings.Builder{}
		log := zerolog.New(buf)

		invalidCollectorError := newInvalidCollectorJobError()
		invalidCollectorError.addInvalidNotifier("a", "b")
		invalidCollectorError.addInvalidNotifier("c", "d")
		invalidCollectorError.addInvalidNotifier("c", "e")
		invalidCollectorError.addInvalidSpec("f", "g")
		invalidCollectorError.addInvalidSpec("f", "h")
		invalidCollectorError.addInvalidSpec("i", "j")

		log.Error().Err(invalidCollectorError).Msg("failed")

		received := InvalidCollectorErrorJson{}
		err := json.Unmarshal([]byte(buf.String()), &zerologMsg{Error: &received})
		require.NoError(t, err)

		require.Equal(t, InvalidCollectorErrorJson{
			NotifiersByJob: map[string][]string{
				"a": {"b"},
				"c": {"d", "e"},
			},
			SpecsByJob: map[string][]string{
				"f": {"g", "h"},
				"i": {"j"},
			},
		}, received, buf.String())
	})
}

func TestCollectorJobValidate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		valid := `
		{
			"checksum": "eyMazDwyR48=",
			"reporting_jobs": {
				"LTCyYoCOrYA=": {
					"checksum": "fg7Nod/Y0Fk=",
					"qr_id": "root",
					"uuid": "LTCyYoCOrYA=",
					"child_jobs": {
						"RzpLCn0YICg=": {}
					}
				},
				"RzpLCn0YICg=": {
					"checksum": "r1RdBtbuknk=",
					"qr_id": "//captain.api.mondoo.app/spaces/test-infallible-taussig-796596",
					"uuid": "RzpLCn0YICg=",
					"child_jobs": {
						"r6ZMtev8wkI=": {}
					},
					"notify": [
						"LTCyYoCOrYA="
					]
				},
				"r6ZMtev8wkI=": {
					"checksum": "1Cz86GvvDxs=",
					"qr_id": "//policy.api.mondoo.app/spaces/test-infallible-taussig-796596/policies/a-policy",
					"uuid": "r6ZMtev8wkI=",
					"child_jobs": {
						"52TcNIfJGm8=": {}
					},
					"notify": [
						"RzpLCn0YICg="
					]
				},
				"52TcNIfJGm8=": {
					"checksum": "bRyDH0EuntQ=",
					"qr_id": "6aqieaqAKlg=",
					"uuid": "52TcNIfJGm8=",
					"notify": [
						"r6ZMtev8wkI="
					],
					"datapoints": {
						"ysE1DRIpkLDWeMqYrNXZCjASfhhznefIkSPtcvu1vy8lZ3X0D6KEx0QQaDeg26zo4KdCKMVrCPQJNH4VoFTsvQ==": true
					}
				}
			}
		}
		`
		collectorJob := &CollectorJob{}
		err := protojson.Unmarshal([]byte(valid), collectorJob)
		require.NoError(t, err)
		err = collectorJob.Validate()

		require.NoError(t, err)
	})

	t.Run("invalid", func(t *testing.T) {
		invalid := `
		{
			"checksum": "eyMazDwyR48=",
			"reporting_jobs": {
				"LTCyYoCOrYA=": {
					"checksum": "fg7Nod/Y0Fk=",
					"qr_id": "root",
					"uuid": "LTCyYoCOrYA=",
					"child_jobs": {
						"RzpLCn0YICg=": {}
					}
				},
				"RzpLCn0YICg=": {
					"checksum": "r1RdBtbuknk=",
					"qr_id": "//captain.api.mondoo.app/spaces/test-infallible-taussig-796596",
					"uuid": "RzpLCn0YICg=",
					"child_jobs": {
						"r6ZMtev8wkI=": {}
					},
					"notify": [
						"LTCyYoCOrYA="
					]
				},
				"52TcNIfJGm8=": {
					"checksum": "bRyDH0EuntQ=",
					"qr_id": "6aqieaqAKlg=",
					"uuid": "52TcNIfJGm8=",
					"notify": [
						"r6ZMtev8wkI="
					],
					"datapoints": {
						"ysE1DRIpkLDWeMqYrNXZCjASfhhznefIkSPtcvu1vy8lZ3X0D6KEx0QQaDeg26zo4KdCKMVrCPQJNH4VoFTsvQ==": true
					}
				}
			}
		}
		`
		collectorJob := &CollectorJob{}
		err := protojson.Unmarshal([]byte(invalid), collectorJob)
		require.NoError(t, err)
		err = collectorJob.Validate()
		invalidCollectorError := newInvalidCollectorJobError()

		require.ErrorAs(t, err, &invalidCollectorError)

		require.Equal(t, map[string][]string{
			"RzpLCn0YICg=": {"r6ZMtev8wkI="},
		}, invalidCollectorError.InvalidSpecsByReportingJob)
		require.Equal(t, map[string][]string{
			"52TcNIfJGm8=": {"r6ZMtev8wkI="},
		}, invalidCollectorError.InvalidNotifiersByReportingJob)
	})
}
