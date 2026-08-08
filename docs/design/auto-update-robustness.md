# Robust auto-update for cnspec & cnquery

**Status:** implemented on `claude/auto-update-robustness-review-5cgcn5` (mql + cnspec), not yet pushed
**Scope:** Linux, macOS, Windows — binary self-update and provider-plugin auto-update, including `serve` mode
**Goal:** in super-critical fleets an update must never leave an agent worse off than before it started — verify before trust, commit atomically, prove the new version runs, and always keep a way back.

---

## 1. Current state (before this change)

There are **two independent update surfaces**, at very different maturity levels.

### Surface A — provider plugins (the always-on unattended path)
In-process, fully ours. `serve.go` calls `updateProviders()` before every scan; every provider start also runs `TryProviderUpdate` (`mql/providers/coordinator.go`). It downloads a `.tar.xz` from `releases.mondoo.com`, unpacks it, and renames the files into place.

Already good:
- Crash-safe extraction: per-file `Sync()`, temp-dir-then-rename, `syncDir()`.
- Resilient transport: `retryablehttp` (3 retries) + idle-timeout reader.

### Surface B — the cnspec/cnquery binary
**Already implemented and enabled on macOS and Linux.** `AutoUpdateEngine` is in `DefaultFeatures` for non-Windows builds (`mql/features_autoupdate.go`), and `cnspec.go` runs `selfupdate.CheckAndUpdate` at startup. The model is a **shadow-bin exec**:

- A newer binary is staged in `~/.config/mondoo/bin/` and the process `exec`s into it on each invocation.
- The package-manager binary in `/usr/bin` is never overwritten — it remains an implicit baseline.
- Windows in-place swap is handled specially (`inplace.go`) to preserve firewall rules.
- Loop prevention: after an update the child runs with `MONDOO_AUTO_UPDATE_ENGINE=false`.

Windows binary self-update is the one piece still gated off (a TODO in `features_autoupdate.go`).

> Because the engine auto-update is already enabled on Mac/Linux, the valuable work here is **hardening the path that is already running**, not adding a new one.

### Gaps that matter for critical fleets

| # | Gap | Surface |
|---|-----|---------|
| 1 | **No integrity/authenticity check on downloads.** Provider path never hashes the archive; binary path checks only the hash embedded in `latest.json` (no signature). | A + B |
| 2 | **No rollback / no "last known good" for providers.** Install overwrites the old files; `installVersion` compares only the *version string*, never launches the new provider. | A |
| 3 | **No health gate before a binary is trusted.** The staged binary is activated after a `version` call succeeds; a binary that starts but crashes under the real workload is not caught. | B |
| 4 | **No crash-loop escape.** If a staged binary starts but crash-loops (worst case: `serve`), every restart re-execs into the same broken binary. | B |
| 5 | **A single bad "latest" rolls the whole fleet.** No pin or floor. | A + B |
| 6 | **Non-atomic multi-file provider swap.** Binary + 2 JSON files rename in three steps; a crash between them mixes a new binary with an old schema. | A |

---

## 2. Design

### 2.1 Shared verification (`mql/utils/verify`)
One package, used by both surfaces. Two independent, composable gates, run in this order:

1. **Signature over the manifest first.** A detached [minisign](https://jedisct1.github.io/minisign/) signature over the `SHA256SUMS` file, verified against a public key pinned into the binary at build time (`verify.PinnedPublicKey`, set via `-ldflags`). Pure Ed25519 + BLAKE2b, no external tool, works air-gapped. Verifying the manifest *before* trusting any digest in it is what turns the checksum into a real authenticity check.
2. **Checksum against the (now-trusted) manifest.** Stream the artifact, compare SHA256 to the manifest entry for that exact filename.

Policy (`verify_signature`): `auto` (verify when a signature exists, else fall back to checksum — but a *present-but-invalid* signature always fails), `require` (fail closed), `off`.

**Artifacts that already exist** (integrity ships today):
- Providers: `releases.mondoo.com/providers/<name>/<version>/<name>_<version>_SHA256SUMS` (`provider_bundler.sh`).
- Binary: `<project>_v<version>_SHA256SUMS` (goreleaser `checksum:`).

**Companion pipeline change needed for signatures** (the one piece outside these two repos): publish `..._SHA256SUMS.sig` (minisign) and set `verify.PinnedPublicKey` at release. Until then, SHA256 protects integrity and signature stays in `auto` (log-only).

### 2.2 Provider install: versioned + health-gated + rollback (`mql/providers`)

New on-disk layout (backward compatible):

```
~/.config/mondoo/providers/<name>/
  .current              # text pointer -> active version
  13.4.1/  <name> <name>.json <name>.resources.json
  13.4.2/  ...
```

State machine per update:

1. **Download** tar.xz + `SHA256SUMS` (+ `.sig`) via the existing retry/idle-timeout client.
2. **Verify** (§2.1). Fail → discard, keep current active.
3. **Stage** into `<name>/.staging-<version>/`, fsync (today's crash-safety kept).
4. **Health-check**: launch the staged binary as `run_as_plugin` and complete the real plugin handshake, then kill it. Fail → discard staging, keep current.
5. **Commit**: rename staging → `<version>/`, then atomically write `.current`. This single rename is the commit point.
6. **Prune** to the last N (default 2) versions, never removing the active or previous.

Rollback is implicit: any failure before the pointer flip leaves the previous version governing. Reader resolves `.current` → version dir, or falls back to the legacy flat layout so existing installs keep working until their next update migrates them.

**Pin/floor** (`ResolveUpdateTarget`): a pinned provider is held at an exact version; a floor refuses any published version below a baseline. One bad "latest" cannot roll a pinned fleet.

### 2.3 Binary self-update: hardened on top of the existing engine (`mql/cli/selfupdate`)

Additions to the already-enabled shadow-bin flow:

- **Signature verification** of the downloaded archive (§2.1), layered on the existing `latest.json` hash check.
- **`selftest` health gate.** Before activating any staged binary — both the "download & install" path and the "exec the already-staged newer binary" path — run `<binary> selftest` (no network, no user data; it initializes features + the provider runtime and exits 0). A binary that fails is **quarantined**: renamed aside and recorded so it is neither activated now nor re-downloaded.
- **Crash-loop startup guard** (`update-state.json` next to the staged binary). Each activation of a not-yet-confirmed version increments a counter; after `maxActivationAttempts` (3) without a healthy confirmation the version is quarantined and the launcher falls back to the package-manager binary. The running binary **confirms** itself once it has demonstrably worked — a CLI command completed, or `serve` finished its first scan cycle — which stops the counter so a good update is never rolled back.

This is the escape hatch that closes gap #4: a broken `serve` binary that starts-then-crashes reverts to the known-good package binary within a few systemd restarts, automatically.

---

## 3. As-built changeset

### mql (`claude/auto-update-robustness-review-5cgcn5`)
- `utils/verify/` — `checksum.go`, `minisign.go`, `verify.go`, `pinnedkey.go` (+ `verify_test.go`)
- `providers/install_layout.go`, `healthcheck.go`, `verify_config.go`, `update_policy.go` (+ `*_test.go`)
- `providers/providers.go`, `registry.go`, `coordinator.go` — versioned install, verification wiring, pin/floor config
- `cli/selfupdate/guard.go`, `verify.go` (+ `guard_test.go`); `selfupdate.go` — health gate + guard wired in

### cnspec (`claude/auto-update-robustness-review-5cgcn5`)
- `apps/cnspec/cmd/selftest.go` — the health-check target
- `apps/cnspec/cmd/serve.go` — apply `update:` config, honor pin/floor, confirm-after-first-scan, opt-in periodic binary self-update
- `apps/cnspec/cmd/config/config.go` — `update:` block + `binary_auto_update`

All touched packages build; new logic is unit-tested (verify gates, layout resolution/prune, pin/floor, crash-loop guard transitions).

---

## 4. Failure & recovery matrix

Invariant: every path ends with the agent running the last known-good version.

| Failure | Detected by | Response | Ends up |
|---|---|---|---|
| Download truncated / mirror error | SHA256 mismatch | discard, retry next tick | running · current |
| Tampered / unsigned artifact (`require`) | signature gate | discard, report | running · current |
| New provider won't hand-shake | provider health-check | previous version stays `current` | running · previous |
| New binary fails selftest | binary health-check | quarantine, don't activate | running · current |
| New binary crash-loops under serve | startup guard + state file | revert to package binary, quarantine version | recovered · baseline |
| Crash mid-install | atomic pointer / rename | pointer never moved; partial dir ignored | running · current |
| Registry unreachable | HTTP client (existing) | retry ×3, skip tick | running · current |
| Bad "latest" published | pin / floor | never selected | running · pinned |

---

## 5. Configuration (`mondoo.yml`)

```yaml
auto_update: true              # providers (existing key, honored)
binary_auto_update: false      # periodic binary self-update *inside serve* (see §7 open question)
update:
  verify_signature: auto       # auto | require | off
  channel: stable              # reserved
  keep_versions: 2             # retained known-good provider versions
  providers:
    os:  { pin: "13.40.4" }    # hold exact version, or…
    aws: { min: "13.2.0" }     # …refuse anything below this floor
```

Secure defaults: checksum always enforced; signature in `auto`; retention 2.

---

## 6. Coordination items (outside these two repos)

1. **Publish the manifest signature.** Sign `SHA256SUMS` with minisign, publish `..._SHA256SUMS.sig`, and set `verify.PinnedPublicKey` via `-ldflags` at release. This flips signature verification from log-only to enforced.
2. **Repo coupling.** cnspec's serve wiring uses new mql APIs. A `go.work` links them for local builds; for release, mql ships first and cnspec's `go.mondoo.com/mql/v13` dependency is bumped (or the commented `replace` is enabled).
3. **Windows engine auto-update** is still gated off (`features_autoupdate.go` TODO). The hardening here applies the moment it is enabled; enabling it is a separate decision.

---

## 7. Open question — `binary_auto_update` default

Because engine auto-update is **already enabled on Mac/Linux**, the new `binary_auto_update` flag governs only the *extra periodic re-check while `serve` runs long* (the startup check already happens regardless). Two reasonable choices:

- **A — follow `auto_update`** (no separate flag): serve periodically re-checks whenever engine auto-update is on, consistent with startup behavior. Simplest mental model; larger blast radius during a long serve.
- **B — separate opt-in flag, default off** (current): the long-running serve does not chase new binaries mid-run unless explicitly told to; it still self-updates at startup like every other command.

Current implementation is **B**. If you prefer the periodic serve behavior to match the already-enabled engine auto-update, switching to **A** is a one-line default change.
