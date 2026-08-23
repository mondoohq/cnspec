// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package forms

// The Databases family: fifteen connectors that share one spec.
//
// This is the only file here whose specs are not written one per connector, and
// that is a fact about the providers rather than a shortcut. They declare one
// flag vocabulary between them, so one spec is the honest description; see
// databaseSpec for why the union of the vocabulary is deliberate and what
// TestEverySpecNamesRealFlags does about it.

// The database providers share one flag vocabulary -- host, port, user,
// database, then a password and an --ask-pass -- so they share one spec rather
// than a dozen near-identical copies. They are the best-behaved family in the
// tree: every one of them marks its password flag FlagOption_Password, so the
// secret classifier needs no help here, and the spec only fixes the ordering.
//
// Their usage strings read "postgresdb [host]" but they declare MinArgs=0 and
// MaxArgs=0: the host is a flag, not a positional. The empty Positional here is
// what suppresses the derived slot.
var databaseSpec = FormSpec{
	Target: []string{
		"host", "port", "instance", "service", "sid", "database",
		"auth-db", "scheme", "organization", "user",
	},
	// --ask-api-key is not named here, and neither is clickhousecloud's
	// --ask-secret. weaviate declares one as a plain bool with no
	// FlagOption_AskInput, which mql parses and nothing reads, so the toggle
	// promised a prompt that never came and the scan ran unauthenticated.
	// genericFields drops both; see isInertPromptFlag.
	Credential: []string{
		"ask-pass", "password", "api-key", "token", "auth",
	},
	Labels: map[string]string{
		"ask-pass":     "prompt for password",
		"tls-insecure": "skip TLS verification",
	},
}

var databaseConnectors = []string{
	"postgresdb", "mysqldb", "mssql", "mongo", "redisdb", "cassandra",
	"elasticsearch", "opensearch", "weaviate", "clickhouse", "clickhousedb",
	"clickhousecloud", "oracledb", "db2", "neon",
}

func init() {
	for _, name := range databaseConnectors {
		registerSpec(name, databaseSpec)
	}
}
