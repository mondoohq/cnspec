# Content validation

Everything that checks the policies in [`content/`](..) lives in this directory. This file is the reference for all of it: what each check proves, when CI runs it, and how to run it yourself.

The policies are the product. A check that compiles but never matches an asset, a compliance tag that points at the wrong control, or a remediation snippet that does not actually fix the thing it claims to fix are all invisible in a green build unless something looks for them. Each area below exists because one of those shipped.

## The map

| Area | The question it answers | How | Where |
|---|---|---|---|
| **Lint** | Does every check compile against the provider schema? | `cnspec policy lint` | no files here |
| **[`scans/`](scans)** | When this check meets this input, does it reach the verdict we claim? | Go, provider-backed | `scans/*_test.go` |
| **[`compliance/`](compliance)** | Are the framework tags internally coherent? | Go, static | `compliance/*_test.go` |
| **[`remediation/code/`](remediation/code)** | Is the fix we ship well-formed in its own language? | Python + each language's linter | `remediation/code/*.py` |
| **[`remediation/commands/`](remediation/commands)** | Do the CLI and API calls we ship actually exist? | Python + CLI grammars / OpenAPI specs | `remediation/commands/*.py` |
| **[`upstream/`](upstream)** | Is what we validate *against* still current? | Python, network | `upstream/*.py` |
| **Spelling** | Typos in policy prose | `typos` | `typos.toml` at the repo root |

Two of these overlap deliberately. `remediation/code/terraform.py` proves a snippet is *well-formed*; `scans`' `TestRemediationSatisfiesCheck` proves it is *right*, by scanning the snippet and requiring the check that recommends it to pass. A snippet can name every property correctly and still demonstrate the exact misconfiguration the check forbids. Both have caught real defects the other could not.

## Directory layout

```
content/validation/
├── README.md                  this file
├── paths.py                   where the validators read from and write to
│
├── scans/                     provider-backed Go suites
│   ├── main_test.go              TestMain: provider provisioning
│   ├── bundles_test.go           whole-bundle smoke scans
│   ├── iac_variants_test.go      per-check pass/fail fixtures  [tag: iac_variants]
│   ├── coverage_test.go          every IaC variant has fixtures [tag: iac_variants]
│   ├── remediation_closes_check_test.go   the fix satisfies its own check [tag: iac_variants]
│   ├── remediation-budget.json   variants whose remediation does not satisfy them yet
│   └── fixtures/
│       ├── iac-variants/<policy>/<check-uid>/{pass,fail}/<scenario>/
│       └── bundles/<policy>-{pass,fail}/
│
├── compliance/                static Go suites, no providers
│   └── owasp_mapping_test.go
│
├── remediation/
│   ├── code/                  one validator per remediation language
│   │   ├── terraform.py  cloudformation.py  bicep.py  ansible.py
│   │   └── powershell.py  bash.py  chef.py
│   └── commands/              CLI and REST API call validation
│       ├── validate.py           entry point and dispatch
│       ├── common.py             shared extraction and reporting
│       ├── cobra.py              kubectl / gh / glab / hcloud / databricks
│       ├── openapi.py            every REST API provider
│       └── aws.py azure.py gcloud.py alicloud.py digitalocean.py nutanix.py oci.py vercel.py
│
├── upstream/                  keeping the above honest
│   ├── pins.py                   the single registry of every pin
│   ├── check.py                  report what has moved
│   ├── bump.py                   rewrite the pins that move mechanically
│   └── dump/                     regenerate the checked-in grammars and specs
│       └── azure.py alicloud.py vercel.py ncli.py api_specs.py
│
└── data/                      checked-in CLI grammars and OpenAPI specs
```

`paths.py` is small but load-bearing. Every validator resolves `CONTENT_DIR`, `DATA_DIR` and `REPO_ROOT` from it rather than counting `.parent` hops from its own file, because a wrong hop does not raise: a `Path` pointing at a directory that does not exist globs cleanly and yields nothing, so a validator aimed at the wrong place finds no policies, reports no failures, and exits 0.

## Running it locally

Every check has a make target. Run them from the repository root.

```bash
make test/content            # everything that runs with nothing installed but Go and cnspec
```

That covers lint, the bundle smoke scans, and the compliance mappings. Three groups are left out of it, each for its own reason, and CI runs all of them:

- `test/content/iac` downloads providers and runs thousands of scans.
- `test/content/remediation` needs each language's linter installed.
- `test/content/commands` needs each cloud's CLI on PATH.

Run the one that covers what you touched.

| Target | Runs | Needs |
|---|---|---|
| `make test/content/lint` | `cnspec policy lint` over `content/` and `content/querypacks` | `cnspec` on PATH |
| `make test/content/scans` | whole-bundle smoke scans | Go |
| `make test/content/compliance` | compliance-tag mappings | Go |
| `make test/content/iac` | all five IaC fixture suites | Go, ~30 min |
| `make test/content/iac/terraform` | one IaC suite (also `/cloudformation`, `/bicep`, `/dockerfile`, `/kubernetes`) | Go |
| `make test/content/iac/coverage` | every IaC variant has pass+fail fixtures | Go, no scans |
| `make test/content/iac/remediation` | each remediation snippet satisfies its own check | Go, ~60 min |
| `make test/content/remediation` | all seven remediation code-block validators | see below |
| `make test/content/remediation/terraform` | one of them (also `/cloudformation`, `/bicep`, `/ansible`, `/powershell`, `/bash`, `/chef`) | that language's linter |
| `make test/content/commands` | CLI and API calls; `CLOUD=aws` scopes it | that cloud's CLI |
| `make test/content/upstream` | which pins are behind | network |
| `make test/content/spelling` | `typos` over the repo | `brew install typos-cli` |

Linters the remediation code validators need: `tflint`, `cfn-lint`, the Bicep CLI, `ansible-lint`, `pwsh`, `shellcheck`, `cookstyle`. Each validator prints an install hint and exits non-zero when its linter is missing, so you never get a silent pass.

### Scoping a run down to one check

The IaC suites name each subtest `<policy>/<check-uid>/<pass|fail>/<scenario>`, so a trailing slash scopes a run to a single check in a couple of seconds:

```bash
go test -tags iac_variants ./content/validation/scans \
  -run 'TestTerraformVariants/mondoo-aws-security/mondoo-aws-security-s3-bucket-encryption-terraform-hcl/'
```

The trailing slash matters. Without it the pattern matches the suite name and runs everything under it.

For the command validators, pass the cloud:

```bash
make test/content/commands CLOUD=aws
python3 content/validation/remediation/commands/validate.py       # every target
python3 content/validation/remediation/commands/validate.py ?     # any unknown name prints the valid ones
```

## When CI runs what

| Workflow | Trigger | Runs | Blocking |
|---|---|---|---|
| `policies_lint.yaml` | PR + push to main, `content/**` | `cnspec policy lint`, uploaded as SARIF | yes, on `error` findings |
| `pr-test-lint.yml` (Code Test) | PR + push | `go test ./...`, which includes `scans` (untagged) and `compliance` | yes |
| `content-iac-tests.yaml` | PR + push to main, `content/**` | the five IaC suites, the coverage gate, and the closed loop, as a matrix | yes |
| `validate-remediation.yaml` | PR + push to main, `content/**` | all seven code-block validators and the command validator, one job each | yes |
| `spell-check.yaml` | PR | `crate-ci/typos` | yes |
| `xgrep.yaml` | PR + push to main | SAST over the repo, fixtures excluded | no, report-only |
| `validation-upstream-drift.yaml` | Mondays 06:00 UTC | pin drift report into one long-lived issue | no |
| `validation-dependency-updates.yaml` | Mondays 05:00 UTC | one PR per kind of pin that moved | no |

`content-iac-tests.yaml` fans out because the Terraform suite is by far the largest. Each provider-backed scan spawns a provider subprocess, and in-process concurrency is capped at a safe `-parallel` to avoid subprocess-contention deadlocks, so the only way to cut wall-clock is more runners. `IAC_SHARD_TOTAL` is the runner count and `IAC_SHARD_INDEX` this runner's 0-based slot; the harness hashes each scenario name with FNV-1a and runs only the ones that land in its shard. The hash is stable across runs and machines, so a scenario is always in the same shard and a rerun of one shard is reproducible. Currently: terraform across 8 runners, the closed loop across 4, everything else on one each.

The fixture-coverage job runs no scans at all. It compares the IaC variants each policy declares against the fixtures on disk, which is why it is cheap enough to gate on.

## The suites in detail

### Lint

`cnspec policy lint` is the first thing to run and the first thing to fix. It checks bundle structure, compiles every `mql:` and `filters:` block against the installed provider schema, and reports deprecated symbols. A check that does not compile cannot be tested by anything below it.

```bash
cnspec policy lint content/mondoo-aws-security.mql.yaml   # one policy
make test/content/lint                                     # everything
```

Deprecated-symbol reporting splits by site: `query-deprecated-symbol` covers `mql:` only, and `filter-deprecated-symbol` fires separately for every `filters:` block at its own line.

### Bundle smoke scans (`scans/bundles_test.go`)

One sample project per policy that should score 100, and one that should score 0, under `scans/fixtures/bundles/`. These run end to end against the whole bundle, so they catch a new check that fires on a clean project. Untagged, so they run in the ordinary `go test ./...`.

On a score mismatch the test logs every check that scored below 100, which is usually enough to name the culprit without rerunning.

### IaC variant fixtures (`scans/iac_variants_test.go`)

The per-check suite. Each `-terraform-hcl`, `-terraform-plan`, `-terraform-state`, `-cloudformation` and `-bicep` variant, plus every check in the native Dockerfile and Kubernetes policies, gets its own fixture directory:

```
scans/fixtures/iac-variants/<policy>/<check-uid>/
├── pass/<scenario>/main.tf      the check must pass
└── fail/<scenario>/main.tf      the check must fail
```

A variant may have any number of pass and fail scenarios; each immediate subdirectory holding a fixture file is one. Fixture files placed directly in `pass/` or `fail/` count as a single scenario.

Three outcomes are distinguished, and the third is the one that matters most:

- **passed** — the check ran and scored 100
- **failed** — the check ran and scored below 100
- **skipped** — the check never ran, because no asset matched its `filters:`

A skipped check is treated as a fixture bug, not a pass. Silently-skipped checks are the failure mode this suite exists to prevent: a variant whose filter never matches anything looks identical to a variant that passes, in every report, forever.

Two markers change the contract, and both must state a reason:

- **`fail/IMPOSSIBLE.md`** — the variant asserts exactly what its own `filters:` require, so no failing input exists. This is the only sanctioned way to ship a variant without a fail fixture, and it counts as fail-covered.
- **`KNOWN_BUG.md`** in a scenario directory — a still-open bug keeps this scenario from asserting correctly. The scenario is kept as a live regression fixture: it passes while the correct outcome is unreachable and **fails once it starts asserting correctly**, which is the signal to delete the marker. A stale marker breaks the build, so adding a check means deleting its markers in the same change.

### Variant coverage (`scans/coverage_test.go`)

Every IaC variant in every registered policy must carry both a pass and a fail fixture. Coverage is at 100%, so this is a flat assertion rather than a debt budget that can grow: a variant added without fixtures fails here instead of merging untested.

```bash
make test/content/iac/coverage
```

### The remediation has to satisfy the check (`scans/remediation_closes_check_test.go`)

This asks the one question the linters cannot: does the fix we ship actually make our own check pass?

For each IaC variant it takes the remediation snippet its parent documents in the same language, writes it into a scenario directory, and scans it with the same machinery the pass/fail fixtures use, requiring the check to pass. The snippet is generated per run rather than checked in, because it is a copy of content that already lives in the policy and a checked-in copy would drift from it silently.

Only what a snippet needs to be a scannable document is added: a `provider "x" {}` stub for HCL, so a filter reading `terraform.providers` does not skip the check, and the `AWSTemplateFormatVersion` preamble for a CloudFormation fragment. HCL snippets also have documentation ellipses and `<placeholder>` tokens substituted out, the same way `remediation/code/terraform.py` does before handing a snippet to tflint. YAML and Bicep snippets are left alone, because there an ellipsis or an angle-bracket token is a *string* and rewriting it could change what the check reads.

Three failures are reported differently, because they mean different things:

- **the check did not run** — the snippet declares no resource the check's filter matches, so applying it as shown leaves the check unevaluated
- **the remediation does not satisfy the check** — a reader who applies the documented fix still fails
- **scanning failed** — the snippet is not a scannable document of its kind on its own

A variant whose remediation does not satisfy it yet is listed in `scans/remediation-budget.json` with a reason. The list may only shrink: an entry that starts passing fails the test, so the entry is removed together with the fix, the same contract as a `KNOWN_BUG.md` marker. An entry naming a variant that no longer has a testable snippet is reported as stale.

A failed scan is retried before it is believed. Under the suite's concurrency a provider subprocess occasionally dies mid-request (`rpc error: code = Unavailable`), which is contention rather than a property of the snippet; a deterministic error reproduces on every attempt and is reported after the last one.

### Compliance tag mappings (`compliance/`)

Static checks over the `compliance/<framework>: <control-uid>` tags the policies carry. They read the bundle files only and run no scans, which is why they live in their own Go package: the `scans` package's `TestMain` provisions cloud providers, which a pure mapping check neither needs nor should depend on.

These guard the mapping against silent drift. They do not and cannot tell you whether a given tag is *correct* — that means reading the framework text for the control. Never copy a `compliance/*` tag from a neighbouring check.

### Remediation code blocks (`remediation/code/`)

Remediation that is *code* rather than a CLI invocation gets linted with the tool its ecosystem already uses. Each validator extracts the fenced blocks from one `- id:` and lints each snippet in isolation, so a snippet has to stand on its own.

| Validator | Reads | Linter |
|---|---|---|
| `terraform.py` | `id: terraform` | tflint, with per-cloud provider rulesets |
| `cloudformation.py` | `id: cloudformation` | cfn-lint |
| `bicep.py` | `id: bicep` | `bicep build` |
| `ansible.py` | `id: ansible` | ansible-lint |
| `powershell.py` | every ```` ```powershell ```` fence, including `audit:` | the PowerShell parser |
| `bash.py` | `id: bash` / `script` / `sh` | shellcheck |
| `chef.py` | `id: chef` | cookstyle |

Each takes an optional target (`linux`, `windows`, `macos`, `kubernetes`, `chef`, …) and `--github-actions`.

**A validator only sees the policies in its `TARGETS`.** Adding a remediation method to a policy that is not listed there means it ships unlinted and CI stays green, so when a policy gains its first `terraform`/`ansible`/`bash` remediation, add it to that validator's `TARGETS` in the same change. `bash.py` is the exception: it has no allowlist, deliberately, because a `TARGETS` list is a standing invitation to exactly that failure.

For Terraform there is a second step: the resource prefix needs an entry in `PROVIDER_MAP` too. Without one the generated `required_providers` block comes out empty and the `terraform_required_providers` rule fails *every* resource in that policy, which looks like 20 content bugs rather than one missing line.

`bash`, `script` and `sh` are three names for one method, so the shell validator reads all three. The **fence language** decides the dialect, which is what keeps a `script` entry holding PowerShell out of shellcheck: only ```` ```bash ```` and ```` ```sh ```` fences are linted.

Some diagnostics are deliberately downgraded or ignored, each for a reason that is about documentation snippets rather than about the language:

- **cfn-lint** — `E1010` is off, because a `!GetAtt` target cannot be stubbed the way an undeclared `!Ref` can (the attribute's type depends on a resource type the snippet never names). A deprecated runtime or engine version (`W2531`, `W3690`) **fails**, since the fix is to bump a string; a whole service being retired (`W3696`, `W3697`) prints as `[INFO]`, since the fix is to migrate the check, which is a content decision.
- **Bicep** — `BCP057` ("the name X does not exist in the current context") is ignored: a remediation example wires itself to a key vault or subnet it does not declare, and declaring the name as `param x object` merely trades `BCP057` for `BCP036`/`BCP240`. `BCP081` and `no-hardcoded-env-urls` print as `[INFO]`.
- **PowerShell** — checks three things in descending order of certainty. The snippet must **parse**. Commands that **resolve on the runner** get their parameter names checked against the real cmdlet. Commands from a module the runner lacks (Az, Microsoft.Graph, VMware.PowerCLI, ExchangeOnlineManagement) get their **name shape** checked instead, since PowerShell requires Verb-Noun with an approved verb and a wrong verb is the usual way to misremember a cmdlet. `<placeholder>` tokens are substituted first, and genuinely dynamic parameters (`Set-ItemProperty -Type`, real on Windows and invisible on the Linux runner) live in `DYNAMIC_PARAMETERS`.

Snippets are dedented out of the `desc: |` block scalar before linting, because Bicep is indentation-sensitive at the token level once a nested object is involved, and YAML always is.

The Bicep job takes several minutes: the CLI reloads the ARM type index on every invocation, so each snippet costs roughly a second and a half.

### Remediation CLI and API commands (`remediation/commands/`)

Verifies that every CLI command and REST call in a remediation section names a real subcommand, a real flag, and a real endpoint. It reads `id: cli` and `id: api` blocks **and `audit:` blocks** — a wrong audit command misleads an auditor exactly as a wrong remediation misleads an operator.

```bash
python3 content/validation/remediation/commands/validate.py            # everything
python3 content/validation/remediation/commands/validate.py aws        # one target
```

Targets: `aws`, `azure`, `oci`, `gcp`, `digitalocean`, `nutanix`, `alicloud`, `vercel`, the Cobra CLIs (`kubernetes`, `github`, `gitlab`, `hetzner`, `databricks`), and the REST APIs (`cloudflare`, `tailscale`, `slack`, `atlassian`, `grafana`, `mongodbatlas`).

Two parsing details follow from reading audit blocks, which are shaped differently from remediation blocks:

- A `$(...)` command substitution is pulled out and validated as a command in its own right, then blanked out of the surrounding text. Left inline, its flags read as flags of the outer command.
- A subcommand held in a shell variable (`for cmd in list-a list-b; do aws sagemaker $cmd …`) has no literal name to check and is accepted rather than reported as unknown.

**Where the command data comes from** decides what you need installed and how it goes stale:

- **Introspected live** (`aws`, `oci`, `gcp`, `digitalocean`, and the Cobra CLIs) — the grammar is built in memory at validation time from the installed CLI, so that CLI must be on PATH. `aws` reads the botocore service models, `oci` walks the Click command tree, `gcp` reads the SDK's static completion tree, `digitalocean` walks `doctl --help`, and the Cobra CLIs walk the hidden `__complete` command with credentials stripped so positional-value completions stay empty.
- **Checked-in grammar** (`azure`, `alicloud`, `vercel`, `nutanix`) — in `data/`, regenerated by `upstream/dump/`. Azure CLI metadata is too slow to refresh every run; `vercel` is a Node.js CLI with no completion surface at all.
- **Pinned OpenAPI download** (`cloudflare`, `slack`, `grafana`, `mongodbatlas`) — fetched at validation time from a raw URL pinned to a commit SHA, cached under `~/.cache/cnspec-validation/`.
- **Checked-in OpenAPI spec** (`tailscale`, `atlassian`, `vercel`) — the vendor serves the spec from a live unversioned endpoint, so it is checked into `data/` and refreshed with `upstream/dump/api_specs.py`.

For REST providers the validator checks the path, the HTTP method, and the `--data` JSON payload against the operation's `requestBody` schema: field names, types, enums, and required properties. Angle-bracket (`<account-name>`) and environment-variable (`$ORG_ID`) placeholders act as wildcards. Known spec-versus-docs divergences are listed per provider under `body_exemptions`; `path_exemptions` allowlists endpoints the API serves but omits from its spec; `strip_api_version` normalizes a leading `/vN` segment for API-first products like Vercel that keep several versions live while the spec documents one per operation.

Two targets are special. `azure` validates **both** `mondoo-azure-security.mql.yaml` and `mondoo-m365-security.mql.yaml`, because the M365 policy's CLI remediations also use `az`. `vercel` runs a CLI validator *and* a REST API validator together, because the Vercel policy fixes some settings with the `vercel` CLI and others with `curl`.

Two things about the Azure grammar are worth knowing before trusting it:

- `upstream/dump/azure.py` runs in two phases. Phase 1 loads the whole command table from the CLI's Python internals, the only practical way to enumerate ~7,000 commands, but its flag names are argparse *destination* names for every argument whose alias is registered globally: `resource_group_name` is exposed as `--resource-group`/`-g`, never as `--resource-group-name`. Phase 2 re-reads `az <cmd> --help` and **replaces** phase 1's flags for the commands the validator will check. Unioning them would keep accepting the invented spelling.
- About a seventh of the grammar comes from **CLI extensions**, and `az` installs none by default. `AZURE_EXTENSIONS` declares the ones the policies use; the script installs any that are missing and refuses to write a grammar if one contributed nothing. Commands from undeclared extensions are dropped, so whatever else happens to be installed cannot change the output.

**Never hand-edit anything in `data/`.** Extend or re-run the dump script instead.

### Spelling

`crate-ci/typos` with the allowlist in `typos.toml` at the repository root. The allowlist is **case-insensitive**, so one lowercase entry covers every casing. Add genuinely new terms to `[default.extend-words]`; where a policy deliberately shows a misspelling as an example, reword the surrounding prose instead.

Fixtures and checked-in grammars are excluded, since they are captured output rather than prose.

## Keeping the pinned upstreams current

Every validator above is only as good as the thing it checks against, and each of those is pinned somewhere.

| Pin | Declared in |
|---|---|
| linter releases (`cfn-lint`, `ansible-lint`, `cookstyle`, `tflint`) | `.github/workflows/validate-remediation.yaml` |
| CLI release artifacts (`bicep`, `doctl`, `glab`, `hcloud`, `databricks`) — version **and** SHA-256 | `.github/workflows/validate-remediation.yaml` |
| tflint ruleset plugins, Terraform provider `~>` constraints | `remediation/code/terraform.py` (`TFLINT_PLUGIN_MAP`, `PROVIDER_MAP`) |
| OpenAPI specs pinned to a commit | `remediation/commands/openapi.py` (`*_OPENAPI_SHA`) |
| checked-in CLI grammars | the `_meta` block of each `data/*.json` |

Dependabot watches `gomod` and `github-actions`. It watches none of the above. `upstream/pins.py` is the single registry: it reads each pin out of the file that declares it, so there is no second copy to go stale. Adding an entry to `PROVIDER_MAP` or `TFLINT_PLUGIN_MAP` puts it under watch automatically; a new linter or CLI needs a line in `WORKFLOW_TOOLS` or `WORKFLOW_CHECKSUMMED`.

```bash
python3 content/validation/upstream/check.py                  # what has moved
python3 content/validation/upstream/check.py --format json    # same, for tooling
python3 content/validation/upstream/bump.py --list            # which pins bump mechanically
python3 content/validation/upstream/bump.py --all --dry-run
python3 content/validation/upstream/bump.py --only cfn-lint   # rewrite one pin in place
python3 content/validation/upstream/bump.py --only terraform-provider  # every behind pin of one kind
python3 content/validation/upstream/bump.py --verify-checksums
```

Regenerating the checked-in data:

```bash
python3 content/validation/upstream/dump/azure.py      # when the Azure CLI version changes
python3 content/validation/upstream/dump/alicloud.py   # from the api.aliyun.com metadata service
python3 content/validation/upstream/dump/vercel.py     # when bumping the pinned vercel CLI
python3 content/validation/upstream/dump/ncli.py       # when bumping the pinned AOS release
python3 content/validation/upstream/dump/api_specs.py  # Tailscale + Atlassian + Vercel specs
```

Two workflows run weekly off this registry. `validation-upstream-drift.yaml` reports the whole table into one long-lived issue. `validation-dependency-updates.yaml` opens one pull request per **kind** of pin that moved, on a stable branch named for the kind (`deps/validation/<kind>`) and not for a version, so next week's run updates the open pull request instead of opening a second one.

Per kind and not per pin, because a kind is declared in one dense block of one file: 24 Terraform providers on 24 consecutive lines of `PROVIDER_MAP`, three rulesets on three lines of `TFLINT_PLUGIN_MAP`, four spec SHAs on four lines of `openapi.py`. Git merges with three lines of context, so a pull request per pin meant every bump of a kind conflicted with every other one the moment the first merged. The trade is that a red bump blocks its group-mates; if one row needs work the rest should not wait on, drop that line from the branch and merge the rest, and the next run offers it again.

Things to know when touching this:

- **A bump is never applied without review.** The tooling does the mechanical half; whether a red CI run means the pin is wrong or the content is remains a judgement call. Closing a bump pull request is a valid answer, and the branch reopens when upstream moves again.
- **A CLI's version and its checksum can never move apart.** The bumper re-downloads the artifact using the URL read back out of the workflow's own `curl` line and recomputes the digest, so it hashes exactly what CI fetches. `--verify-checksums` re-derives all of them at their *current* pins.
- **Terraform provider entries hold a constraint, not a version.** `~> 5.0` floats across all of 5.x, so a provider only outgrows it on a major; below 1.0 the minor is the breaking axis, so `~> 0.111` outgrows on a minor. A provider major bump is a content migration, not a dependency bump.
- **Grammars are not string bumps.** The pin records which tool produced the checked-in JSON, so it moves by installing that tool and re-running the dump script. `--all` reports them as skipped rather than pretending.
- **`upstream/dump/vercel.py` is the one place a version is written twice** (its `VERCEL_VERSION` constant and the JSON's `_meta`). `bump.py --sync-dump-pins` pulls the constant back into line after a regeneration.

## Adding a check: what has to pass

1. `make test/content/lint` — it compiles.
2. If it has IaC variants, add pass **and** fail fixtures under `scans/fixtures/iac-variants/<policy>/<check-uid>/`, or a `fail/IMPOSSIBLE.md` stating why no failing input exists. `make test/content/iac/coverage` enforces this.
3. If it ships remediation in a language with a validator, make sure the policy is in that validator's `TARGETS` (and `PROVIDER_MAP`, for Terraform). Otherwise it ships unlinted and CI stays green.
4. `make test/content/iac/terraform -run` scoped to your check — the fixture asserts what you think it does, and is not silently *skipped*.
5. If the check recommends an IaC fix, the closed loop will scan that snippet and require the check to pass. Run it before you find out in CI.
6. Delete any `KNOWN_BUG.md` marker your change fixes, in the same change. A stale marker fails the build.

See [`../CLAUDE.md`](../CLAUDE.md) for the authoring rules: bundle structure, variants, compliance tags, and MQL idioms.

## Adding a new validator

Put it in the area that matches the question it answers, not the language it is written in. Then:

- Resolve paths through `paths.py`. Never count `.parent` hops.
- Add a `make test/content/...` target, so there is exactly one way to run it.
- Add a row to **The map** and to **When CI runs what** in this file.
- If it checks against anything versioned, register the pin in `upstream/pins.py` in the same change. An unregistered pin is a validator that will keep passing while checking against something that no longer exists.
