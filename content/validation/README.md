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
| **[`live/`](live)** | Does this check reach the verdict we claim, against the real database? | Python + Docker | `live/*.py` |
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
│   └── fixtures/
│       ├── iac-variants/<policy>/<check-uid>/{pass,fail}/<scenario>/
│       └── bundles/<policy>-{pass,fail}/
│
├── compliance/                static Go suites, no providers
│   └── owasp_mapping_test.go
│
├── live/                      runtime database policies, scanned in Docker
│   ├── verify.py                 entry point, dispatch, and the coverage gate
│   ├── common.py                 fixture model, docker driver, cnspec scan
│   └── redisdb.py cassandra.py clickhousedb.py
│                                 the fixtures and their expected verdicts
│
├── remediation/
│   ├── code/                  one validator per remediation language
│   │   ├── terraform.py  cloudformation.py  bicep.py  ansible.py
│   │   └── powershell.py  bash.py  chef.py
│   └── commands/              CLI and REST API call validation
│       ├── validate.py           entry point and dispatch
│       ├── common.py             shared extraction and reporting
│       ├── cobra.py              kubectl / gh / glab / hcloud / databricks / stackit
│       ├── openapi.py            every REST API provider
│       └── aws.py azure.py gcloud.py alicloud.py digitalocean.py nutanix.py
│           oci.py openstack.py proxmox.py vercel.py
│
├── upstream/                  keeping the above honest
│   ├── pins.py                   the single registry of every pin
│   ├── check.py                  report what has moved
│   ├── bump.py                   rewrite the pins that move mechanically
│   └── dump/                     regenerate the checked-in grammars and specs
│       └── azure.py alicloud.py vercel.py ncli.py proxmox.py api_specs.py
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
- `test/content/live` needs Docker, and starts real database servers.

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
| `make test/content/live` | the runtime database policies, against real servers | Docker, cnspec, ~6 min |
| `make test/content/live/redisdb` | one of them (also `/cassandra`, `/clickhousedb`) | Docker, cnspec |
| `make test/content/upstream` | which pins are behind | network |
| `make test/content/upstream/unit` | the pin resolvers, against recorded payloads | none |
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

`content-iac-tests.yaml` fans out because the Terraform suite is by far the largest. Each provider-backed scan spawns a provider subprocess, and in-process concurrency is capped at a safe `-parallel` to avoid subprocess-contention deadlocks, so the only way to cut wall-clock is more runners. `IAC_SHARD_TOTAL` is the runner count and `IAC_SHARD_INDEX` this runner's 0-based slot; the harness hashes each scenario name with FNV-1a and runs only the ones that land in its shard. The hash is stable across runs and machines, so a scenario is always in the same shard and a rerun of one shard is reproducible.

Every suite goes through the same `runFixtureSuite`, so every one of them shards; the counts differ only because the suites differ in size. Currently terraform runs across 12 runners, the closed loop across 6, cloudformation across 4, bicep across 3, and dockerfile and kubernetes on one each. Size a count from measured scan time rather than from the number of variants: a job spends about 55 seconds on checkout, Go and the cache restores before it reaches the first scan, so a shard carrying under roughly two minutes of scans spends more of its life starting up than testing. (That floor used to be nearer two minutes: TestMain's seven provider installs were about 57 seconds of it until the workflow started caching them.) The number to divide is the per-shard `Run <suite> variant tests` step time in a recent run, minus that shared prep.

The fixture-coverage job runs no scans at all. It compares the IaC variants each policy declares against the fixtures on disk, which is why it is cheap enough to gate on.

`live/` is the one area with no workflow. It needs a Docker daemon and pulls several database images, and the Cassandra fixtures put a floor of a few minutes on any run, so it is run locally when one of the three runtime database policies changes. See [Live database scans](#live-database-scans-live) for what that buys and what wiring it up would cost.

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

Every variant in the corpus satisfies its own remediation, so this is a flat assertion rather than a shrink-only debt budget, the same shape the fixture-coverage gate took once it reached 100%. A snippet that stops closing its check fails here instead of being excused by an entry on a list.

The first two failures usually share one cause: a snippet that documents only the fixing resource. A check's `filters:` select on the resource being protected, so a fix shown without its subject leaves the check unevaluated. Where the rest of the policy matches nothing else in the snippet either, no policy binds to the asset at all and the scan errors rather than reporting a skip, which surfaces as the third failure instead.

The remaining cause is a value the HCL parser cannot resolve statically. A `jsonencode` body collapses to an empty list if any leaf inside it is a resource reference, and a policy supplied through an `aws_iam_policy_document` data source arrives as the reference string. Either reads as absent rather than as the value it becomes at apply time, so a snippet whose correctness lives inside one has to spell that part out literally.

A failed scan is retried before it is believed. Under the suite's concurrency a provider subprocess occasionally dies mid-request (`rpc error: code = Unavailable`), which is contention rather than a property of the snippet; a deterministic error reproduces on every attempt and is reported after the last one.

### Live database scans (`live/`)

Three policies in `content/` assess a *running server* rather than a file: `mondoo-redis-security`, `mondoo-cassandra-security`, and `mondoo-clickhousedb-security`. Nothing else in this directory can reach them. The IaC suites scan checked-in source; the bundle smoke scans need a scannable project; lint proves only that a query compiles against the provider schema. A check that compiles and reads the wrong field is invisible to every other gate here — it ships, and it is wrong on every asset.

So this suite starts the real database in Docker, brings it to a known configuration, runs the shipped policy against it, and requires each check to reach the verdict the fixture claims.

```bash
make test/content/live                  # all three
make test/content/live/redisdb          # one of them
python3 content/validation/live/verify.py cassandra --keep   # leave containers up
```

Fixture modules are discovered, not registered: every `.py` in `live/` that is not `verify.py` or `common.py` is a database suite, and each exports `build_suite(workdir)`. A registry would be the thing someone forgets — a module sitting in the directory looking complete and never running.

The model is two types deep:

```
Fixture   one container, brought to a configuration
  Scan    one `cnspec scan` against it, with the verdict expected per check
```

A fixture carries several scans when the states differ by a few statements. The Cassandra `secured` fixture is scanned twice — once hardened, once after a handful of statements walk it backwards — because starting a second Cassandra costs ninety seconds and running the statements costs one.

**Expectations are exhaustive, and `error` is one of them.** A scan that returns a check the fixture makes no claim about is a failure, so a new check cannot appear without someone deciding what it does against a live server. `error` is a real expectation rather than a tolerated failure: the ClickHouse 24.8 fixture expects four checks to error, because the provider cannot decode `system.users.auth_type` on that line. When that is fixed, the fixture fails and someone comes back here.

**Both sides, or a written reason.** The coverage gate requires every check in the policy to be observed passing *and* failing across the fixtures. A check that is only ever seen failing is a check nobody has proven can be satisfied, which is exactly how a permanently-failing check ships; a check only ever seen passing has not been shown to discriminate. An `error` counts as neither. Where a single-container fixture genuinely cannot reach a side, the reason goes in `no_pass_fixture` or `no_fail_fixture` on the suite and is printed on every run — an exemption nobody reads is an exemption that stops being a decision. An exemption for a check that no longer exists fails the gate.

Three Cassandra checks are exempted on the pass side: client and internode encryption need JKS keystores built into the image, and a `system_auth` replication factor of three needs three nodes. That last one is worth knowing before you try it — setting a replication factor above the node count makes the `LOCAL_QUORUM` read Cassandra performs during authentication unsatisfiable, and locks every account out of the cluster.

**What the fixtures have caught.** Both of these compiled, linted, and read plausibly:

- `mondoo-clickhousedb-security-secure-tcp-port-enabled` filtered `serverSettings` for `tcp_port_secure`. That setting is not in `system.server_settings` on any ClickHouse version, so the query was `[].any()` — false on every server, forever. The check was removed; restore it when the provider can answer through `getServerPort()`.
- `mondoo-redis-security-plaintext-port-disabled` read `redisdb.instance.port == 0`. Redis reports `tcp_port` in `INFO` for whichever listener is active, so on a server with `port 0` and a TLS listener — the exact configuration the check told operators to adopt — the provider returned 6379 and the check failed. `CONFIG GET port` returns 0 there. The check was removed; restore it when `redisdb.instance.port` reads from CONFIG.

Neither was reachable without running the policy against a real server in the state its own remediation recommends.

**Fixture details that are load-bearing**, each learned by watching it fail:

- Readiness is a query round trip, not an open port. Cassandra binds 9042 well before `system_auth` holds the default superuser, and a scan in that window fails to authenticate for reasons that have nothing to do with the policy.
- The hardened Redis fixture sits on its own Docker network with a fixed address. A container on the default bridge is assigned an address only at start, so a configuration file baked beforehand cannot name one, and `bind` would have to stay a wildcard — leaving the bound-address check with no pass fixture at all.
- The ClickHouse users file is named `zz-cnspec-live.xml`. Files in `users.d` merge in filename order, and the official image writes its own `default-user.xml` there; a name sorting before it lets that fragment re-add a `default` account with no authentication method, and the server refuses to start.
- The hardened ClickHouse fixture writes a bootstrap account in, uses it to create the quota, and writes it back out before scanning. An account with access management is the only way to create a quota, and it would then be the one thing failing the least-privilege check.
- Cassandra audit logging is set in `cassandra.yaml`, not with `nodetool enableauditlog`. `nodetool` changes the running state without touching `system_views.settings`, which is where the provider reads, so the check stays failing on a node that is demonstrably auditing.

**Not wired into CI.** It needs a Docker daemon and pulls several database images, and the Cassandra fixtures put a floor of a few minutes on any run. Run it locally when you touch one of the three policies. Wiring it up is a reasonable next step — the suite exits nonzero on any mismatch and prints in the same `[PASS]`/`[FAIL]` shape as the other Python validators — but that is a decision about runner minutes rather than about the harness.

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
| `powershell.py` | every ```` ```powershell ```` fence in every policy, including `audit:` | the PowerShell parser |
| `bash.py` | `id: bash` / `script` / `sh`, in every policy | shellcheck |
| `chef.py` | `id: chef` | cookstyle |

Each takes an optional target (`linux`, `windows`, `macos`, `kubernetes`, `chef`, …) and `--github-actions`.

**A validator only sees the policies in its `TARGETS`.** Adding a remediation method to a policy that is not listed there means it ships unlinted and CI stays green, so when a policy gains its first `terraform`/`ansible`/`chef` remediation, add it to that validator's `TARGETS` in the same change.

**`bash.py` and `powershell.py` are the exceptions, deliberately.** Both default to every policy in `content/`, because a `TARGETS` list is a standing invitation to exactly that failure and neither linter needs per-policy setup: shellcheck lints a shell script, and the PowerShell parser parses PowerShell, wherever the fence appears. Their `TARGETS` dicts name groups for running one area locally (`bash.py linux`) and do not define what CI covers, so a new policy needs no entry in either.

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

For `aws` it also checks that the command supplies every parameter the operation **requires**. Naming only real flags is not enough: a snippet missing a required one is rejected by argument parsing before it reaches AWS, so the documented fix cannot run at all. Members carrying `idempotencyToken` are excluded, because botocore fills those in, and `AWS_CLI_DEFAULTED_PARAMS` in `commands/aws.py` lists the few the CLI itself defaults. Confirm a command really is rejected before adding an entry there — argument validation is client-side, so dummy credentials are enough to tell a `ParamValidation` error from one that means the command parsed.

```bash
python3 content/validation/remediation/commands/validate.py            # everything
python3 content/validation/remediation/commands/validate.py aws        # one target
```

Targets: `aws`, `azure`, `oci`, `gcp`, `digitalocean`, `nutanix`, `alicloud`, `openstack`, `proxmox`, `vercel`, the Cobra CLIs (`kubernetes`, `github`, `gitlab`, `hetzner`, `databricks`, `stackit`), and the REST APIs (`cloudflare`, `tailscale`, `slack`, `atlassian`, `grafana`, `mongodbatlas`, `okta`, `portainer`).

A provider whose fixes are all UI click-through still belongs here. `okta` and `portainer` have no CLI for the settings their checks target, so their entire API surface is `audit:` steps — and a wrong audit command is exactly as misleading as a wrong remediation.

Two parsing details follow from reading audit blocks, which are shaped differently from remediation blocks:

- A `$(...)` command substitution is pulled out and validated as a command in its own right, then blanked out of the surrounding text. Left inline, its flags read as flags of the outer command.
- A subcommand held in a shell variable (`for cmd in list-a list-b; do aws sagemaker $cmd …`) has no literal name to check and is accepted rather than reported as unknown.
- Pipelines are split at the shell operator, and the split is quote-aware. A CloudWatch metric filter pattern such as `'{ ($.eventName = A) || ($.eventName = B) }'` contains a `||` that is data, not a pipeline; splitting on it would truncate the command and leave every later flag unchecked.

**Where the command data comes from** decides what you need installed and how it goes stale:

- **Introspected live** (`aws`, `oci`, `gcp`, `digitalocean`, `openstack`, and the Cobra CLIs) — the grammar is built in memory at validation time from the installed CLI, so that CLI must be on PATH. `aws` reads the botocore service models, `oci` walks the Click command tree, `gcp` reads the SDK's static completion tree, `digitalocean` walks `doctl --help`, `openstack` reads the completion script `openstack complete` prints, and the Cobra CLIs walk the hidden `__complete` command with credentials stripped so positional-value completions stay empty.
- **Checked-in grammar** (`azure`, `alicloud`, `vercel`, `nutanix`, `proxmox`) — in `data/`, regenerated by `upstream/dump/`. Azure CLI metadata is too slow to refresh every run; `vercel` is a Node.js CLI with no completion surface at all. `proxmox` is the one target that needs no CLI installed anywhere: Proxmox VE generates its API, its API viewer and its CLI tools from one JSON Schema, and `pvesh` takes API paths directly, so the published schema validates it outright.
- **Pinned OpenAPI download** (`cloudflare`, `slack`, `grafana`, `mongodbatlas`) — fetched at validation time from a raw URL pinned to a commit SHA, cached under `~/.cache/cnspec-validation/`.
- **Checked-in OpenAPI spec** (`tailscale`, `atlassian`, `vercel`, `okta`, `portainer`) — checked into `data/` and refreshed with `upstream/dump/api_specs.py`, for one of two reasons. The first three are *unpinnable*: the vendor serves the spec from a live unversioned endpoint that can change without notice. The last two are *YAML*: the spec is pinned to an exact upstream revision, but the validators are stdlib-only and cannot parse YAML at validation time, so it is converted once and the revision recorded in the file's `_meta`.

For REST providers the validator checks the path, the HTTP method, and the `--data` JSON payload against the operation's `requestBody` schema: field names, types, enums, and required properties. Angle-bracket (`<account-name>`) and environment-variable (`$ORG_ID`) placeholders act as wildcards. Known spec-versus-docs divergences are listed per provider under `body_exemptions`; `path_exemptions` allowlists endpoints the API serves but omits from its spec; `strip_api_version` normalizes a leading `/vN` segment for API-first products like Vercel that keep several versions live while the spec documents one per operation.

Two targets are special. `azure` validates **both** `mondoo-azure-security.mql.yaml` and `mondoo-m365-security.mql.yaml`, because the M365 policy's CLI remediations also use `az`. `vercel` runs a CLI validator *and* a REST API validator together, because the Vercel policy fixes some settings with the `vercel` CLI and others with `curl`.

Two things about the Azure grammar are worth knowing before trusting it:

- `upstream/dump/azure.py` runs in two phases. Phase 1 loads the whole command table from the CLI's Python internals, the only practical way to enumerate ~7,000 commands, but its flag names are argparse *destination* names for every argument whose alias is registered globally: `resource_group_name` is exposed as `--resource-group`/`-g`, never as `--resource-group-name`. Phase 2 re-reads `az <cmd> --help` and **replaces** phase 1's flags for the commands the validator will check. Unioning them would keep accepting the invented spelling.
- About a seventh of the grammar comes from **CLI extensions**, and `az` installs none by default. `AZURE_EXTENSIONS` declares the ones the policies use; the script installs any that are missing and refuses to write a grammar if one contributed nothing. Commands from undeclared extensions are dropped, so whatever else happens to be installed cannot change the output.

OpenStack has the same shape of problem as those Azure extensions, one layer up. OpenStackClient ships the core services and nothing else; `loadbalancer`, `coe`, `database`, `datastore` and `secret` each come from a separate PyPI package that registers its own cliff entry points, and `openstack complete` describes only what is installed. A missing package does not shrink the grammar visibly — it turns 22 real commands into "unknown command". `OPENSTACK_PLUGINS` in `commands/openstack.py` declares the ones the policy needs, and the validator exits rather than build a grammar when one of them contributed nothing.

That module also rebuilds the tree by walking down from the root rather than by splitting the completion script's keys. The keys join path words with `_`, which is unambiguous only for as long as no command name contains one; walking down means each level's names come from its parent's value list, so the grammar stays correct whatever a future command is called.

Two things about the Proxmox grammar are worth knowing:

- It covers **`pvesh` only**. `pct`, `qm`, `pveum`, `pvenode`, `pvesm` and `pvecm` are generated from the same schema, but by Perl code that maps each subcommand to an API path, and that mapping is not published in machine-readable form — pve-docs generates the man pages from it at build time and does not check the generated synopsis files in. Validating those would mean hand-writing the subcommand-to-path map, which is inventing the oracle rather than reading one, so they are left unchecked deliberately rather than checked against something made up here.
- It reads `- id: bash` remediation as well as `- id: cli`, because this policy documents each fix twice: the single command under `cli`, and the same fix wrapped in a node-loop script under `bash`. Reading only `cli` left 12 of its 29 `pvesh` invocations unchecked. `extract_bash_blocks` defaults to `("cli",)`, so any policy that puts vendor CLI calls under another method id needs that id passed explicitly — shellcheck lints those blocks, but nothing there knows whether the command exists.

**Never hand-edit anything in `data/`.** Extend or re-run the dump script instead.

**Every dump is written pretty-printed, and that is a reviewability rule rather than a stylistic one.** Nothing reads these files but the validators, so formatting cannot change what they accept. What it changes is the one place a human is in the loop: the weekly refresh pull request, whose entire purpose is to show which endpoints, commands and schemas moved. A multi-megabyte document on a single line renders as one changed line, which GitHub declines to display at all, so approving the bump degrades into trusting the vendor. `upstream/data_format_test.py` fails the build on a single-line dump, so a script that starts minifying is caught rather than merged. The scripts differ on `indent=1` versus `indent=2` and that is fine; what is not allowed is one line.

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
| YAML-only OpenAPI specs, converted and checked in | `upstream/dump/api_specs.py` (`OKTA_SPEC_VERSION`, `PORTAINER_SPEC_SHA`) |
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
python3 content/validation/upstream/dump/proxmox.py    # when bumping the pinned pve-docs commit
python3 content/validation/upstream/dump/api_specs.py            # every checked-in spec
python3 content/validation/upstream/dump/api_specs.py okta       # just one, to keep the diff small
```

Two workflows run weekly off this registry. `validation-upstream-drift.yaml` reports the whole table into one long-lived issue. `validation-dependency-updates.yaml` opens one pull request per **kind** of pin that moved, on a stable branch named for the kind (`deps/validation/<kind>`) and not for a version, so next week's run updates the open pull request instead of opening a second one.

Per kind and not per pin, because a kind is declared in one dense block of one file: 24 Terraform providers on 24 consecutive lines of `PROVIDER_MAP`, three rulesets on three lines of `TFLINT_PLUGIN_MAP`, four spec SHAs on four lines of `openapi.py`. Git merges with three lines of context, so a pull request per pin meant every bump of a kind conflicted with every other one the moment the first merged. The trade is that a red bump blocks its group-mates; if one row needs work the rest should not wait on, drop that line from the branch and merge the rest, and the next run offers it again.

Things to know when touching this:

- **A bump is never applied without review.** The tooling does the mechanical half; whether a red CI run means the pin is wrong or the content is remains a judgement call. Closing a bump pull request is a valid answer, and the branch reopens when upstream moves again.
- **A CLI's version and its checksum can never move apart.** The bumper re-downloads the artifact using the URL read back out of the workflow's own `curl` line and recomputes the digest, so it hashes exactly what CI fetches. `--verify-checksums` re-derives all of them at their *current* pins.
- **Terraform provider entries hold a constraint, not a version.** A pin is behind when the newest release is one the constraint cannot resolve, not when it is spelled differently to what this repo would write from scratch. Terraform's pessimistic operator lets only the rightmost named component increment, so `~> 5.0` floats across all of 5.x, `~> 0.111` across the rest of 0.x, and a deliberately narrow `~> 1.288` across every later 1.x. A provider major bump is a content migration, not a dependency bump.
- **Only *released* versions become constraints.** The Terraform Registry's `version` field is the newest version *published*, prereleases included, and terraform will not select a prerelease for a `~>` constraint, so a constraint derived from one resolves to nothing at all. That shipped twice: `aliyun/alicloud` published a 2.0 beta while its released line was 1.x, the bump proposed `~> 2.0`, and tflint never noticed because it does not resolve versions. Every resolver now ends in `stable_token`, and the Terraform one reads the registry's version list rather than that field. Both rules are covered by `make test/content/upstream/unit`.
- **Grammars are not string bumps.** The pin records which tool produced the checked-in JSON, so it moves by installing that tool and re-running the dump script. `--all` reports them as skipped rather than pretending.
- **`upstream/dump/vercel.py` is the one place a version is written twice** (its `VERCEL_VERSION` constant and the JSON's `_meta`). `bump.py --sync-dump-pins` pulls the constant back into line after a regeneration.

## Adding a policy: what to register

A new check inherits the coverage its policy already has. A new *policy* inherits nothing. Most validators here are allowlist-driven, so a `content/*.mql.yaml` that nobody registers is linted, spell-checked, scanned by `bash.py` and checked by the compliance suites, and examined by nothing else — the bundle ships with its variants untested and its remediation unverified, with every gate green.

This has already happened. PR #3338 added four SaaS policies; a follow-up commit in the same PR registered them with `remediation/code/terraform.py` and that immediately failed **8 of their 11** HCL snippets against the real provider schemas, and hand-checking their CLI snippets found invented `hcp vault` / `hcp consul` command groups and two `neonctl` flags that do not exist. None of it was visible until the policy was on a list.

Register the policy everywhere it applies, in the same change that adds it:

| Where | When it applies | What happens if you skip it |
|---|---|---|
| `content/README.md` | always — a row under the matching platform heading | the policy is invisible in the user-facing catalog |
| `scans/iac_variants_test.go` → `tfVariantPolicies` | the policy has any `-terraform-hcl`, `-terraform-plan`, `-terraform-state`, `-cloudformation` or `-bicep` variant | variants are never scanned, and coverage never asks for fixtures |
| `scans/iac_variants_test.go` → `init()`'s `extraProviders` | its runtime variant needs a provider outside `terraform`/`k8s`/`aws`/`azure`/`gcp`/`cloudformation` (the base list in `main_test.go`) | parallel subtests race to auto-install it and lose with `cannot find resource for identifier '<provider>'` |
| `remediation/code/terraform.py` → `TARGETS` **and** `PROVIDER_MAP` for each resource prefix | any `- id: terraform` remediation | snippets are never resolved against the provider schema; with `TARGETS` but no `PROVIDER_MAP` entry, `required_providers` comes out empty and `terraform_required_providers` fails *every* resource in the policy |
| `remediation/code/{cloudformation,bicep,ansible,chef}.py` → `TARGETS` | that language appears in a remediation block | same, for that language |
| `remediation/commands/` → `CLI_VALIDATORS`, `COBRA_CLIS`, or `API_PROVIDERS` | any `- id: cli` / `- id: api` block, or an `audit:` step that invokes a CLI or REST call | invented commands and endpoints ship unchallenged |
| `live/` → a fixture module named for the provider | the policy assesses a running server rather than a file | no gate ever runs the checks against the thing they describe |
| `compliance/owasp_mapping_test.go` → `llmAnchoredPolicies` | the policy is AI/LLM-focused | nothing guards its OWASP Top 10 for LLM Applications tags against silently going missing |
| `typos.toml` → `[default.extend-words]` | the vendor's terminology trips the spell checker | `spell-check.yaml` fails |

A policy in `PROVIDER_MAP` needs a real provider source and a version constraint that resolves; a constraint matching only prereleases silently fails to resolve.

`remediation/code/bash.py` and `remediation/code/powershell.py` are the exceptions on purpose and need no entry — both glob every policy in `content/`, because a `TARGETS` list is a standing invitation to exactly this failure, and the comment on each one's `TARGETS` says so. The compliance suites glob too (`../../mondoo-*.mql.yaml`), so tag coherence is covered from the first commit.

### Group filters and variants do not mix

A policy whose groups carry `filters: asset.platform == "<api-platform>"` cannot have IaC variants. The group filter is evaluated before the check's own filter, so a Terraform asset never reaches the variant and every fixture reports as *skipped* rather than pass or fail. Policies with variants leave their groups unfiltered and let each variant's `filters:` select its asset — that is why `mondoo-tailscale-security` and `mondoo-snowflake-security` declare no group filters.

The failure mode is loud once you have fixtures (`check did not run against …`) and invisible before that, so convert the groups in the same change that adds the first variant.

### Registering with the command validators

Which registry depends on how the vendor's interface is reached, and anything that introduces a new binary needs a CI change too:

- **A new REST API** — add an entry to `API_PROVIDERS` in `commands/openapi.py` with the policy file, host, and spec source. A spec pinned to a commit SHA also needs its constant registered in `upstream/pins.py`; a YAML-only spec is converted once by `upstream/dump/api_specs.py` and checked into `data/`.
- **A new Cobra CLI** — add an entry to `COBRA_CLIS` in `commands/cobra.py` (CLI name, policy list, `include_audit`, install hint). The CLI must also be installed by the `validate-remediation.yaml` commands job, pinned by version **and** SHA-256.
- **A new cloud CLI with its own grammar source** — a module beside `aws.py`/`azure.py`/`gcloud.py` plus an entry in `CLI_VALIDATORS` in `commands/validate.py`, and the same CI install step.
- **An existing validator gaining a second policy** — some validators name their policies in module constants rather than a registry (`azure.py` validates both `mondoo-azure-security` and `mondoo-m365-security`, because the M365 policy's CLI remediations also use `az`). Add the file there.

When no registry fits because the vendor has no non-interactive surface at all, that is a finding worth recording rather than a gap to leave silent: say so in a comment on the check, the way the Netlify MFA check does, so the next reader does not re-derive it.

## Adding a check: what has to pass

1. `make test/content/lint` — it compiles.
2. If it has IaC variants, add pass **and** fail fixtures under `scans/fixtures/iac-variants/<policy>/<check-uid>/`, or a `fail/IMPOSSIBLE.md` stating why no failing input exists. `make test/content/iac/coverage` enforces this.
3. If it ships remediation in a language with a validator, make sure the policy is in that validator's `TARGETS` (and `PROVIDER_MAP`, for Terraform). Otherwise it ships unlinted and CI stays green.
4. `make test/content/iac/terraform -run` scoped to your check — the fixture asserts what you think it does, and is not silently *skipped*.
5. If the check recommends an IaC fix, the closed loop will scan that snippet and require the check to pass. Run it before you find out in CI.
6. Delete any `KNOWN_BUG.md` marker your change fixes, in the same change. A stale marker fails the build.
7. If the policy has a `live/` suite — Redis, Cassandra, ClickHouse — add the check to the `expect` map of every fixture, and make sure some fixture sees it pass and some fixture sees it fail. `make test/content/live/<database>` enforces both.

See [`../CLAUDE.md`](../CLAUDE.md) for the authoring rules: bundle structure, variants, compliance tags, and MQL idioms.

## Adding a new validator

Put it in the area that matches the question it answers, not the language it is written in. Then:

- Resolve paths through `paths.py`. Never count `.parent` hops.
- Add a `make test/content/...` target, so there is exactly one way to run it.
- Add a row to **The map** and to **When CI runs what** in this file.
- If it checks against anything versioned, register the pin in `upstream/pins.py` in the same change. An unregistered pin is a validator that will keep passing while checking against something that no longer exists.
