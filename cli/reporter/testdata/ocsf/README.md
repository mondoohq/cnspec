# Compiled OCSF schemas

`schema-<version>.json.gz` is the **compiled** OCSF schema for one version, used by
`TestOcsfSchemaValidation` (see `cli/reporter/ocsf_validate_test.go`) to validate every event
the OCSF reporter emits with the OCSF project's own validator,
[`github.com/ocsf/ocsf-toolkit`](https://github.com/ocsf/ocsf-toolkit).

The files are checked in so the test is hermetic: it needs no network and no Python toolchain.
There is one file per version listed in `ocsf.SupportedVersions()`; adding a version to that
list means adding its schema here.

## Regenerating

The compiled format comes from the [OCSF Schema Compiler](https://pypi.org/project/ocsf-schema-compiler/),
which needs Python 3.14 or newer. `uv` can supply one:

```bash
uv python install 3.14
uv tool install ocsf-schema-compiler --python 3.14

VERSION=1.3.0
git clone --depth 1 --branch "v${VERSION}" https://github.com/ocsf/ocsf-schema.git "ocsf-schema-v${VERSION}"
ocsf-schema-compiler "ocsf-schema-v${VERSION}" | gzip -9 > "schema-${VERSION}.json.gz"
```

Regenerate only when adding a supported version or when the compiler's output format changes;
a released OCSF version is immutable, so its compiled schema does not drift.
