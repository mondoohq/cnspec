# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
"""Docker fixtures and scan plumbing shared by the live database suites.

The policies for Redis, Cassandra, and ClickHouse are the only ones in
`content/` that assess a *running server* rather than a file. Nothing else in
`content/validation/` can reach them: the IaC suites scan checked-in source
files, and lint only proves a query compiles. A query that compiles against the
provider schema and reads the wrong field is invisible to every other gate here.

So this suite starts the real database in Docker, puts it into a known state,
runs the shipped policy against it, and requires each check to reach the verdict
we claim it reaches. Both verdicts: a check that can only be seen failing is a
check nobody has proven can pass, which is how a permanently-failing check ships.

The model is deliberately small:

    Fixture   one container, brought to a known configuration
      Scan    one `cnspec scan` against it, with the verdict expected per check

A fixture may carry several scans when the states differ only by a few CQL or
SQL statements — starting a second Cassandra costs ninety seconds, and running
the seed statements costs one.
"""

from __future__ import annotations

import json
import os
import posixpath
import re
import secrets
import shutil
import subprocess
import sys
import tempfile
import time
from dataclasses import dataclass, field
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))  # content/validation
from paths import CONTENT_DIR  # noqa: E402

# Every container, network, and volume this suite creates carries this prefix,
# so a crashed run can be cleaned up with a single docker command and a stray
# container is recognisable as ours rather than the operator's.
PREFIX = "cnspec-live"

# How long a fixture's readiness probe may take before the run gives up.
# Cassandra dominates this: a cold JVM plus the system_auth bootstrap is
# routinely ninety seconds on a laptop and slower in CI.
READY_TIMEOUT = 300


# --------------------------------------------------------------------------
# setup steps
# --------------------------------------------------------------------------
#
# A fixture reaches its configuration through an ordered list of these. They
# exist as separate types rather than as one "run this shell script" step
# because the failure messages differ: an Exec that fails names the command, a
# WaitFor that times out names what it was waiting for, and a Restart failing is
# an infrastructure problem rather than a fixture problem.


@dataclass(frozen=True)
class Exec:
    """Run a command inside the container. Fails the fixture on a nonzero exit
    unless `may_fail` is set, which is for statements that are idempotent in
    intent but not in SQL (`CREATE ROLE` without `IF NOT EXISTS`)."""

    argv: list[str]
    may_fail: bool = False


@dataclass(frozen=True)
class WaitFor:
    """Poll a command inside the container until it exits 0.

    This is the readiness probe, and it is a command rather than a port check on
    purpose. Cassandra accepts connections on 9042 well before `system_auth` is
    populated, so a port check returns ready and the first scan then fails
    authentication for reasons that have nothing to do with the policy.
    """

    argv: list[str]
    timeout: int = READY_TIMEOUT
    interval: int = 5


@dataclass(frozen=True)
class Restart:
    """Restart the container, then wait for it to report running again.

    Several settings under test cannot be changed at runtime — Cassandra's
    authenticator, ClickHouse's user files — so the fixture writes the
    configuration and restarts into it.
    """

    settle: int = 3


@dataclass(frozen=True)
class Remove:
    """Delete a file inside the container.

    Used to take a bootstrap account back out once it has done its job: the
    ClickHouse hardened fixture needs an account with access management to
    create a quota, and that account would then be the one thing failing the
    least-privilege check.
    """

    path: str


@dataclass(frozen=True)
class Write:
    """Write a file inside the container, creating parent directories."""

    path: str
    content: str


Step = Exec | WaitFor | Restart | Remove | Write


# --------------------------------------------------------------------------
# fixtures and scans
# --------------------------------------------------------------------------


@dataclass(frozen=True)
class Scan:
    """One scan of a fixture, and the verdict every check must reach.

    `expect` maps a check uid to `pass`, `fail`, or `error`. `error` is a real
    expectation rather than a tolerated failure: it records a provider defect we
    know about and want to be told about the moment it is fixed, so the harness
    fails when an `error` expectation starts passing.

    Expectations are exhaustive. A check present in the policy and absent from
    every scan's `expect` is reported by the coverage gate, so adding a check
    without deciding what it does against a live server is not something that
    can happen quietly.
    """

    name: str
    expect: dict[str, str]
    user: str | None = None
    password: str | None = None
    tls: bool = False
    seed: list[Step] = field(default_factory=list)


@dataclass(frozen=True)
class Fixture:
    """One container, brought to a known configuration, then scanned."""

    name: str
    image: str
    container_port: int
    host_port: int
    scans: list[Scan]
    env: dict[str, str] = field(default_factory=dict)
    command: list[str] | None = None
    memory: str | None = None
    # A fixed address, for the one check that asks what the server is bound to.
    # A container on the default bridge gets an address only at start, so a
    # configuration file cannot name it; a user-defined network can.
    network: str | None = None
    ip: str | None = None
    # Files written into the container before it first starts, as
    # container path -> contents. Mounted rather than copied, because some of
    # them are read before the entrypoint would let us in.
    mounts: dict[str, str] = field(default_factory=dict)
    setup: list[Step] = field(default_factory=list)

    @property
    def container(self) -> str:
        return f"{PREFIX}-{self.name}"


@dataclass(frozen=True)
class Suite:
    """Every fixture for one policy, plus what the coverage gate may excuse."""

    provider: str
    policy: str
    policy_uid: str
    fixtures: list[Fixture]
    # uid -> why a live fixture cannot reach that side. Anything here is printed
    # on every run: an exemption that stops being read stops being a decision.
    no_pass_fixture: dict[str, str] = field(default_factory=dict)
    no_fail_fixture: dict[str, str] = field(default_factory=dict)

    @property
    def bundle(self) -> Path:
        return CONTENT_DIR / self.policy


# --------------------------------------------------------------------------
# docker driver
# --------------------------------------------------------------------------


def _run(argv: list[str], timeout: int = 300) -> subprocess.CompletedProcess:
    return subprocess.run(argv, capture_output=True, text=True, timeout=timeout)


def docker_available() -> str | None:
    """Returns None when Docker can run containers, or a reason it cannot."""
    if shutil.which("docker") is None:
        return "docker is not on PATH"
    probe = _run(["docker", "info", "--format", "{{.ServerVersion}}"], timeout=30)
    if probe.returncode != 0:
        detail = probe.stderr.strip().splitlines()
        return f"docker is installed but not reachable: {detail[0] if detail else 'no daemon'}"
    return None


def ensure_network(name: str, subnet: str) -> None:
    if _run(["docker", "network", "inspect", name], timeout=30).returncode == 0:
        return
    created = _run(["docker", "network", "create", "--subnet", subnet, name])
    if created.returncode != 0:
        raise RuntimeError(f"could not create docker network {name}: {created.stderr.strip()}")


def destroy(fixture: Fixture) -> None:
    _run(["docker", "rm", "-f", fixture.container], timeout=120)


def start(fixture: Fixture, workdir: Path) -> None:
    """Create the container and bring it to the fixture's configuration."""
    destroy(fixture)

    argv = ["docker", "run", "-d", "--name", fixture.container]
    argv += ["-p", f"{fixture.host_port}:{fixture.container_port}"]
    if fixture.memory:
        argv += ["-m", fixture.memory]
    if fixture.network:
        argv += ["--network", fixture.network]
    if fixture.ip:
        argv += ["--ip", fixture.ip]
    for key, value in fixture.env.items():
        argv += ["-e", f"{key}={value}"]
    for container_path, contents in fixture.mounts.items():
        host_path = workdir / fixture.name / container_path.lstrip("/").replace("/", "_")
        host_path.parent.mkdir(parents=True, exist_ok=True)
        host_path.write_text(contents)
        # World-readable: the database runs as its own uid inside the container
        # and the file arrives owned by whoever ran the harness.
        host_path.chmod(0o644)
        argv += ["-v", f"{host_path}:{container_path}"]
    argv.append(fixture.image)
    if fixture.command:
        argv += fixture.command

    created = _run(argv, timeout=600)
    if created.returncode != 0:
        raise RuntimeError(f"{fixture.name}: docker run failed: {created.stderr.strip()}")

    apply_steps(fixture, fixture.setup)


def apply_steps(fixture: Fixture, steps: list[Step]) -> None:
    for step in steps:
        if isinstance(step, Exec):
            done = _run(["docker", "exec", fixture.container] + step.argv, timeout=300)
            if done.returncode != 0 and not step.may_fail:
                raise RuntimeError(
                    f"{fixture.name}: {' '.join(step.argv)} exited {done.returncode}: "
                    f"{(done.stderr or done.stdout).strip().splitlines()[:1]}"
                )
        elif isinstance(step, Write):
            # Staged on the host and copied in, rather than piped through a
            # shell inside the container. Nothing here needs a shell, and not
            # having one means no argument can ever be read as syntax.
            parent = posixpath.dirname(step.path)
            if parent:
                _run(["docker", "exec", fixture.container, "mkdir", "-p", parent], timeout=60)
            with tempfile.NamedTemporaryFile("w", delete=False) as staged:
                staged.write(step.content)
                staged_path = staged.name
            try:
                os.chmod(staged_path, 0o644)
                done = _run(["docker", "cp", staged_path, f"{fixture.container}:{step.path}"], timeout=120)
            finally:
                os.unlink(staged_path)
            if done.returncode != 0:
                raise RuntimeError(f"{fixture.name}: could not write {step.path}: {done.stderr.strip()}")
        elif isinstance(step, Remove):
            _run(["docker", "exec", fixture.container, "rm", "-f", step.path], timeout=60)
        elif isinstance(step, Restart):
            done = _run(["docker", "restart", fixture.container], timeout=300)
            if done.returncode != 0:
                raise RuntimeError(f"{fixture.name}: restart failed: {done.stderr.strip()}")
            time.sleep(step.settle)
        elif isinstance(step, WaitFor):
            deadline = time.monotonic() + step.timeout
            while True:
                probe = _run(["docker", "exec", fixture.container] + step.argv, timeout=120)
                if probe.returncode == 0:
                    break
                if time.monotonic() > deadline:
                    raise RuntimeError(
                        f"{fixture.name}: not ready after {step.timeout}s "
                        f"(waiting on `{' '.join(step.argv)}`)"
                    )
                time.sleep(step.interval)
        else:  # pragma: no cover - the union above is closed
            raise TypeError(f"unknown setup step: {step!r}")


# --------------------------------------------------------------------------
# scanning
# --------------------------------------------------------------------------

# cnspec reports a check by its MRN; the uid is the last path segment.
_UID_FROM_MRN = re.compile(r"[^/]+$")


def scan(suite: Suite, fixture: Fixture, scan_spec: Scan) -> dict[str, str]:
    """Run the shipped policy against the fixture, returning uid -> status."""
    argv = [
        "cnspec", "scan", suite.provider,
        "--host", "127.0.0.1",
        "--port", str(fixture.host_port),
        "-f", str(suite.bundle),
        "--policy", suite.policy_uid,
        "-o", "json",
        "--incognito",
    ]
    if scan_spec.user:
        argv += ["--user", scan_spec.user]
    if scan_spec.password:
        argv += ["--password", scan_spec.password]
    if scan_spec.tls:
        argv += ["--tls", "--tls-insecure"]

    done = _run(argv, timeout=900)
    # cnspec writes progress to stderr and the report to stdout, and exits
    # nonzero when checks fail — which is the normal case here, so the exit code
    # is not the signal. An unparseable stdout is.
    try:
        report = json.loads(done.stdout)
    except json.JSONDecodeError:
        raise RuntimeError(
            f"{fixture.name}/{scan_spec.name}: cnspec produced no report "
            f"(exit {done.returncode}): {(done.stderr or done.stdout).strip()[-600:]}"
        )

    statuses: dict[str, str] = {}
    for asset_scores in report.get("scores", {}).values():
        for mrn, score in asset_scores.get("values", {}).items():
            uid = _UID_FROM_MRN.search(mrn).group(0)
            if uid.startswith(suite.policy_uid + "-"):
                statuses[uid] = score.get("status", "unknown")
    if not statuses:
        raise RuntimeError(
            f"{fixture.name}/{scan_spec.name}: the scan returned no check results "
            f"(exit {done.returncode}): {(done.stderr or done.stdout).strip()[-600:]}"
        )
    return statuses


# --------------------------------------------------------------------------
# fixture assets
# --------------------------------------------------------------------------


def credential() -> str:
    """A password for a fixture account, generated fresh on every run.

    The fixtures need accounts, and accounts need passwords, but a password
    written into this directory is a credential in the repository — which
    secret scanners flag, correctly, since they cannot tell a throwaway from a
    real one. Generating them removes the question: there is nothing to leak,
    nothing to rotate, and no allowlist entry that would also hide a real
    credential added later.
    """
    return secrets.token_hex(16)


def mint_certs(workdir: Path) -> dict[str, str]:
    """A CA and a server certificate, returned as PEM text keyed by file name.

    Minted per run rather than checked in, so nothing here expires. It protects
    nothing — the scan connects with --tls-insecure — so its only requirement is
    that the server will load it.
    """
    def openssl(*argv: str) -> None:
        done = subprocess.run(["openssl", *argv], capture_output=True, text=True, cwd=workdir)
        if done.returncode != 0:
            raise RuntimeError(f"openssl {' '.join(argv)} failed: {done.stderr.strip()}")

    openssl("req", "-x509", "-newkey", "rsa:2048", "-sha256", "-days", "3650", "-nodes",
            "-keyout", "ca.key", "-out", "ca.crt", "-subj", "/CN=cnspec-live-ca")
    openssl("req", "-newkey", "rsa:2048", "-nodes",
            "-keyout", "server.key", "-out", "server.csr", "-subj", "/CN=localhost")
    openssl("x509", "-req", "-in", "server.csr", "-CA", "ca.crt", "-CAkey", "ca.key",
            "-CAcreateserial", "-out", "server.crt", "-days", "3650", "-sha256")
    return {name: (workdir / name).read_text() for name in ("ca.crt", "server.crt", "server.key")}


# --------------------------------------------------------------------------
# the checks a policy declares
# --------------------------------------------------------------------------

_CHECK_UID = re.compile(r"^  - uid: (\S+)\s*$")


def declared_checks(suite: Suite) -> set[str]:
    """Every check uid in the bundle, read from the file rather than from a scan.

    Reading the file is what makes the coverage gate work: a check that the
    fixtures never exercise does not appear in any scan result, so a gate built
    on scan output could never notice it was missing.
    """
    uids = set()
    for line in suite.bundle.read_text().splitlines():
        found = _CHECK_UID.match(line)
        if found and found.group(1).startswith(suite.policy_uid + "-"):
            uids.add(found.group(1))
    return uids
