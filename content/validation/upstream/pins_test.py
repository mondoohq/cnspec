#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
"""Unit tests for the pin resolvers.

Every resolver here reads an upstream index and turns it into the string this
repo pins, so the failure mode is not a crash: it is a bump that looks
reasonable and is wrong. A version nothing resolves to, or a pin reported as
behind itself. Both have shipped, twice each, and the only reader downstream is
whoever reviews the pull request the weekly workflow opens.

These resolvers otherwise run on a schedule only, so a defect introduced here
is not observable until that run. The tests are offline -- anything that would
reach a registry patches `fetch_json` with a recorded payload -- so they cost
nothing and run on every pull request that touches `content/`.

    make test/content/upstream/unit
"""

import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest import mock

import pins


class StableTokenTest(unittest.TestCase):
    """A pre-release is never a version a pin may be moved to."""

    def test_recognises_the_two_spellings(self):
        # Semver puts the marker after a hyphen; RubyGems puts it in a dotted
        # segment. Both are "something past the numeric prefix has a letter".
        self.assertTrue(pins.is_prerelease("2.0.0-beta2"))
        self.assertTrue(pins.is_prerelease("0.27.0.beta4"))
        self.assertTrue(pins.is_prerelease("1.0.0-rc1"))

    def test_a_release_is_not_a_prerelease(self):
        self.assertFalse(pins.is_prerelease("1.289.0"))
        # A leading v is stripped before the test, so a tag is not misread as
        # a pre-release on the strength of its own prefix.
        self.assertFalse(pins.is_prerelease("v1.2.3"))

    def test_prerelease_resolves_to_unknown(self):
        # "unknown" is the same fail-safe an unparseable value already gets: it
        # reports as unchecked, which opens no pull request.
        self.assertEqual(pins.stable_token("2.0.0-beta2"), "unknown")
        self.assertEqual(pins.stable_token("1.289.0"), "1.289.0")

    def test_every_resolver_ends_in_stable_token(self):
        # The registries disagree about whether "latest" hides a pre-release,
        # so the rule is applied at the resolver rather than remembered per
        # upstream. One recorded pre-release payload per resolver.
        for resolver, payload, arg in [
            (pins.latest_github_release, {"tag_name": "v9.0.0-beta.1"}, "owner/repo"),
            (pins.latest_pypi, {"info": {"version": "2.0.0b1"}}, "pkg"),
            (pins.latest_rubygem, {"version": "0.27.0.beta4"}, "gem"),
            (pins.latest_npm, {"version": "5.0.0-next.3"}, "pkg"),
        ]:
            with self.subTest(resolver=resolver.__name__):
                with mock.patch.object(pins, "fetch_json", return_value=payload):
                    self.assertEqual(resolver(arg), "unknown")


class NewestStableTest(unittest.TestCase):
    """Picking a release out of a registry's version list."""

    def test_orders_numerically_not_lexically(self):
        # The lists arrive sorted, but sorted as strings, which puts 1.9.0
        # above 1.289.0 and would report a current pin behind.
        self.assertEqual(pins.newest_stable(["1.10.0", "1.289.0", "1.9.0"]), "1.289.0")

    def test_skips_prereleases(self):
        self.assertEqual(
            pins.newest_stable(["1.288.0", "1.289.0", "2.0.0-beta1", "2.0.0-beta2"]),
            "1.289.0",
        )

    def test_nothing_stable_is_empty(self):
        self.assertEqual(pins.newest_stable(["2.0.0-beta1"]), "")
        self.assertEqual(pins.newest_stable([]), "")


class LatestTerraformProviderTest(unittest.TestCase):
    """The registry's `version` field is the newest version *published*."""

    def resolve(self, payload: object) -> str:
        with mock.patch.object(pins, "fetch_json", return_value=payload):
            return pins.latest_terraform_provider("aliyun/alicloud")

    def test_ignores_prereleases(self):
        # aliyun/alicloud as the registry served it in August 2026: a 2.0 beta
        # published while the released line was still 1.x. Taking `version`
        # verbatim produced `~> 2.0`, which matches no released version, so
        # terraform could not resolve the provider at all. tflint never noticed,
        # because it does not resolve versions. The full `versions` list rides
        # along on the same response, so reading it costs no extra request.
        self.assertEqual(
            self.resolve({
                "version": "2.0.0-beta2",
                "versions": ["1.287.0", "1.288.0", "1.289.0", "2.0.0-beta1", "2.0.0-beta2"],
            }),
            "1.289.0",
        )

    def test_orders_the_list_numerically(self):
        self.assertEqual(
            self.resolve({"version": "1.9.0", "versions": ["1.10.0", "1.289.0", "1.9.0"]}),
            "1.289.0",
        )

    def test_falls_back_to_the_version_field(self):
        # A registry response carrying no list still resolves, and still only
        # through `stable_token`.
        self.assertEqual(self.resolve({"version": "1.5.0"}), "1.5.0")

    def test_prerelease_only_provider_is_unknown(self):
        # Nothing to pin to. Reported as unchecked, which opens no pull request
        # -- the right answer for a provider with no release at all.
        self.assertEqual(
            self.resolve({"version": "2.0.0-beta1", "versions": ["2.0.0-beta1"]}),
            "unknown",
        )

    def test_unreachable_registry_is_unknown(self):
        self.assertEqual(self.resolve(None), "unknown")


class LatestGitLabReleaseTest(unittest.TestCase):
    """GitLab has no "exclude pre-releases" flag on the releases endpoint."""

    def resolve(self, payload: object) -> str:
        with mock.patch.object(pins, "fetch_json", return_value=payload):
            return pins.latest_gitlab_release("group%2Fproject")

    def test_takes_the_first_stable_tag_not_the_first_tag(self):
        # Releases come back newest-first, so a pre-release at the top would be
        # the answer if the page were not scanned.
        self.assertEqual(
            self.resolve([{"tag_name": "v1.65.0-rc1"}, {"tag_name": "v1.64.0"}]),
            "1.64.0",
        )

    def test_a_page_with_no_release_is_unknown(self):
        self.assertEqual(self.resolve([{"tag_name": "v1.0.0-rc1"}]), "unknown")
        self.assertEqual(self.resolve([]), "unknown")


class ConstraintAdmitsTest(unittest.TestCase):
    """A `~>` pin is behind when it cannot resolve the newest release."""

    def test_narrow_pin_still_floats_across_its_line(self):
        # The regression. `~> 1.288` covers every 1.x from 1.288 up, so 1.289.0
        # needs no action. Comparing constraint *text* instead reported the pin
        # behind the `~> 1.0` this repo writes from scratch, and had the bump
        # workflow offering to loosen a constraint that is narrow on purpose.
        self.assertTrue(pins.constraint_admits("~> 1.288", "1.289.0"))
        self.assertTrue(pins.constraint_admits("~> 6.0", "6.60.0"))

    def test_a_new_major_outgrows_the_pin(self):
        self.assertFalse(pins.constraint_admits("~> 1.288", "2.0.0"))
        self.assertFalse(pins.constraint_admits("~> 5.0", "6.60.0"))

    def test_only_the_rightmost_named_component_increments(self):
        # `~> 1.0.4` names a patch, so it stops at 1.1.0.
        self.assertTrue(pins.constraint_admits("~> 1.0.4", "1.0.9"))
        self.assertFalse(pins.constraint_admits("~> 1.0.4", "1.1.0"))

    def test_a_zero_x_pin_holds_until_one_point_oh(self):
        # `~> 0.111` really does resolve 0.112.0, so rewriting it to `~> 0.112`
        # would change which provider terraform selects not at all. It outgrows
        # at 1.0 and not before.
        self.assertTrue(pins.constraint_admits("~> 0.111", "0.112.0"))
        self.assertFalse(pins.constraint_admits("~> 0.111", "1.0.0"))

    def test_a_release_below_the_floor_is_not_admitted(self):
        self.assertFalse(pins.constraint_admits("~> 1.288", "1.287.0"))

    def test_a_constraint_this_cannot_parse_is_not_admitted(self):
        # Anything but `~>` falls through to `constraint_for`, which is the
        # conservative direction: it reports rather than staying quiet.
        self.assertFalse(pins.constraint_admits(">= 1.0", "1.2.3"))


class CheckTerraformPinsTest(unittest.TestCase):
    """The provider rows the weekly bump workflow reads."""

    SOURCE = '''
TFLINT_PLUGIN_MAP = {
}

PROVIDER_MAP = {
    "alicloud": ("aliyun/alicloud", "~> 1.288"),
    "aws": ("hashicorp/aws", "~> 5.0"),
    "stackit": ("stackitcloud/stackit", "~> 0.111"),
    "time": ("hashicorp/time", "~> 0.14"),
    "unreleased": ("vendor/unreleased", "~> 1.0"),
}
'''

    UPSTREAM = {
        "aliyun/alicloud": "1.289.0",
        "hashicorp/aws": "6.60.0",
        "stackitcloud/stackit": "0.112.0",
        "hashicorp/time": "1.0.0",
        "vendor/unreleased": "unknown",
    }

    def states(self, upstream: dict[str, str] | None = None) -> dict[str, tuple[str, str]]:
        versions = upstream or self.UPSTREAM
        with TemporaryDirectory() as tmp:
            path = Path(tmp) / "terraform.py"
            path.write_text(self.SOURCE)
            with mock.patch.object(pins, "TERRAFORM_VALIDATOR", path), \
                 mock.patch.object(pins, "latest_terraform_provider", versions.get):
                return {p.name: (p.state, p.latest) for p in pins.check_terraform_pins()}

    def test_narrow_pin_is_current(self):
        # The whole point of the narrowed alicloud pin: it holds the 1.x line
        # while 2.x is beta-only, and reopens nothing meanwhile.
        self.assertEqual(
            self.states()["terraform-provider-alicloud"], ("current", "~> 1.288")
        )

    def test_narrow_pin_still_bumps_once_the_next_line_releases(self):
        # And the pin is not simply pinned forever: a real 2.0.0 outgrows it.
        upstream = {**self.UPSTREAM, "aliyun/alicloud": "2.0.0"}
        self.assertEqual(
            self.states(upstream)["terraform-provider-alicloud"], ("behind", "~> 2.0")
        )

    def test_a_genuinely_outgrown_pin_still_reports(self):
        self.assertEqual(self.states()["terraform-provider-aws"], ("behind", "~> 6.0"))

    def test_zero_x_pin_is_current_within_its_line(self):
        self.assertEqual(
            self.states()["terraform-provider-stackit"], ("current", "~> 0.111")
        )

    def test_zero_x_pin_outgrows_at_one_point_oh(self):
        self.assertEqual(self.states()["terraform-provider-time"], ("behind", "~> 1.0"))

    def test_an_unresolvable_provider_opens_no_bump(self):
        # "unchecked" rather than "behind": there is no version to move to, and
        # a row that cannot be resolved must not be turned into a pull request.
        self.assertEqual(
            self.states()["terraform-provider-unreleased"], ("unchecked", "unknown")
        )


if __name__ == "__main__":
    unittest.main()
