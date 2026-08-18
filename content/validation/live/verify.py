#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Runs the runtime database policies against the real database in Docker and
# requires every check to reach the verdict the fixtures claim it reaches.
#
# Usage:
#   python3 content/validation/live/verify.py            # every database
#   python3 content/validation/live/verify.py redisdb    # one of them
#   python3 content/validation/live/verify.py ?          # list the names
#   python3 content/validation/live/verify.py --keep     # leave containers up
#
# Needs Docker and cnspec on PATH. It pulls database images and starts real
# servers, so it is not part of `make test/content`; see the README.

from __future__ import annotations

import argparse
import importlib
import subprocess
import sys
import tempfile
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))  # content/validation

import common  # noqa: E402
from paths import REPO_ROOT  # noqa: E402

# Fixture modules are discovered rather than listed. Every `.py` beside this one
# that is not part of the harness is a database suite, and each exports
# `build_suite(workdir)`. Discovery rather than a registry because a registry is
# the thing someone forgets: a fixture module added without an entry would sit
# in the directory looking complete and never run.
_HARNESS_MODULES = {"verify", "common"}


def discover() -> dict[str, object]:
    """Fixture modules by database name, in a stable order."""
    here = Path(__file__).resolve().parent
    names = sorted(
        path.stem for path in here.glob("*.py")
        if path.stem not in _HARNESS_MODULES and not path.stem.startswith("_")
    )
    modules = {}
    for name in names:
        module = importlib.import_module(name)
        if not hasattr(module, "build_suite"):
            raise RuntimeError(
                f"{name}.py sits in live/ but exports no build_suite(workdir); "
                "either it is a fixture module and needs one, or it belongs elsewhere"
            )
        modules[name] = module
    return modules


# --------------------------------------------------------------------------
# reporting
# --------------------------------------------------------------------------


class Report:
    """Accumulates results and prints them in the shape the other validators use.

    Every line is flushed. A Cassandra fixture is ninety seconds of silence
    before its first result, and a run redirected to a file would otherwise show
    nothing at all until it finished — indistinguishable from a hang.
    """

    def __init__(self) -> None:
        self.passed = 0
        self.failures: list[str] = []

    def check(self, label: str, ok: bool, detail: str = "") -> None:
        if ok:
            self.passed += 1
            print(f"[PASS] {label}", flush=True)
        else:
            self.failures.append(f"{label}: {detail}")
            print(f"[FAIL] {label}", flush=True)
            if detail:
                print(f"       {detail}", flush=True)

    def note(self, label: str, detail: str) -> None:
        print(f"[SKIP] {label}", flush=True)
        print(f"       {detail}", flush=True)


def verify_suite(suite: common.Suite, report: Report, workdir: Path, keep: bool) -> None:
    print(f"\n=== {suite.policy} ===", flush=True)
    declared = common.declared_checks(suite)
    seen: dict[str, set[str]] = {}

    for fixture in suite.fixtures:
        try:
            common.start(fixture, workdir)
            for scan_spec in fixture.scans:
                if scan_spec.seed:
                    common.apply_steps(fixture, scan_spec.seed)
                actual = common.scan(suite, fixture, scan_spec)
                label_prefix = f"{fixture.name}/{scan_spec.name}"

                unexpected = sorted(set(actual) - set(scan_spec.expect))
                if unexpected:
                    report.check(
                        f"{label_prefix}/expectations-complete", False,
                        "the scan returned checks this fixture makes no claim about: "
                        + ", ".join(unexpected),
                    )

                for uid, want in sorted(scan_spec.expect.items()):
                    got = actual.get(uid, "not-scanned")
                    report.check(
                        f"{label_prefix}/{uid}", got == want,
                        f"expected {want}, got {got}",
                    )
                    seen.setdefault(uid, set()).add(got)
        except Exception as err:  # noqa: BLE001 - a fixture failure is a result
            report.check(f"{fixture.name}/fixture", False, str(err))
        finally:
            if not keep:
                common.destroy(fixture)

    coverage(suite, declared, seen, report)


def coverage(suite: common.Suite, declared: set[str], seen: dict[str, set[str]], report: Report) -> None:
    """Every check must be exercised, and from both sides unless excused.

    This is the part that makes the suite hold. Without it, a check added to the
    policy is simply never scanned: no fixture mentions it, nothing fails, and
    the gate stays green over a check nobody has ever seen run.
    """
    for uid in sorted(declared):
        outcomes = seen.get(uid, set())
        if not outcomes:
            report.check(
                f"coverage/{uid}", False,
                "no fixture in content/validation/live exercises this check",
            )
            continue

        excuse = suite.no_pass_fixture.get(uid)
        if "pass" in outcomes:
            report.check(f"coverage/{uid}/pass", True)
        elif excuse:
            report.note(f"coverage/{uid}/pass", excuse)
        else:
            report.check(
                f"coverage/{uid}/pass", False,
                "never observed passing. A check no fixture can satisfy is the shape of a "
                "check that cannot be satisfied at all — add a fixture, or record why not "
                "in no_pass_fixture",
            )

        # An `error` is not a fail: it means the check never reached a verdict.
        # Counting it here would let a check that only ever errors look covered.
        excuse = suite.no_fail_fixture.get(uid)
        if "fail" in outcomes:
            report.check(f"coverage/{uid}/fail", True)
        elif excuse:
            report.note(f"coverage/{uid}/fail", excuse)
        else:
            report.check(
                f"coverage/{uid}/fail", False,
                "never observed failing, so nothing proves it discriminates — add a fixture, "
                "or record why not in no_fail_fixture",
            )

    # An exemption for a check that no longer exists is a stale excuse, and a
    # stale excuse is how a real gap gets excused later by accident.
    exempted = set(suite.no_pass_fixture) | set(suite.no_fail_fixture)
    for uid in sorted(exempted - declared):
        report.check(
            f"coverage/{uid}/exemption", False,
            "exempted in this suite but no longer present in the policy; remove the entry",
        )


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("target", nargs="?", default="all",
                        help="redisdb, cassandra, clickhousedb, all, or ? to list")
    parser.add_argument("--keep", action="store_true",
                        help="leave the containers running for inspection")
    args = parser.parse_args()

    modules = discover()
    if args.target == "?" or (args.target not in modules and args.target != "all"):
        print("targets: " + ", ".join(["all", *sorted(modules)]))
        return 0 if args.target == "?" else 1

    with tempfile.TemporaryDirectory(prefix="cnspec-live-") as tmp:
        workdir = Path(tmp)
        unavailable = common.docker_available()
        if unavailable:
            print(f"live database verification needs Docker: {unavailable}")
            print("Install Docker Desktop or start the daemon, then re-run.")
            return 1
        if subprocess.run(["which", "cnspec"], capture_output=True).returncode != 0:
            print("live database verification needs cnspec on PATH (make cnspec/install)")
            return 1

        targets = sorted(modules) if args.target == "all" else [args.target]

        # A fixture module declares any Docker network its fixtures need; only
        # the ones about to run are created.
        for name in targets:
            for network, subnet in getattr(modules[name], "NETWORKS", {}).items():
                common.ensure_network(network, subnet)

        report = Report()
        for name in targets:
            verify_suite(modules[name].build_suite(workdir), report, workdir, args.keep)

        print(f"\n{report.passed} passed, {len(report.failures)} failed")
        for failure in report.failures:
            print(f"  {failure}")
        if args.keep:
            print(f"\nContainers left running; remove them with:\n"
                  f"  docker rm -f $(docker ps -aq --filter name={common.PREFIX})")
        print(f"\nPolicies under test live in {REPO_ROOT / 'content'}")
        return 1 if report.failures else 0


if __name__ == "__main__":
    sys.exit(main())
