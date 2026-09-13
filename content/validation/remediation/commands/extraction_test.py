# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Unit tests for the prose extraction in common.py.
#
# The rest of this directory is tested by running it: a parser bug shows up
# as a failure against real policy content. Inline extraction is the
# exception, because its bugs are silent in both directions. Extract too
# little and a wrong flag ships with every gate green, which is the defect
# that motivated it (#3858, #3893). Extract too much and the first run
# buries a reviewer in prose that was never a command.
#
# Run: python3 -m unittest discover -s content/validation/remediation/commands -p "*_test.py"

import unittest

from common import extract_bash_blocks, extract_command_sources, extract_inline_commands

POLICY = """policies:
  - uid: example-policy
    groups:
      - title: Checks
        filters: asset.platform == "openstack"
        checks:
          - uid: example-check-one
          - uid: example-check-two
queries:
  - uid: example-check-one
    title: Example one
    mql: |
      container.readAcl == null
    docs:
      desc: |
        This check verifies that anonymous read is off.

        The ACL is set with `swift post --read-acl ''`, not with
        `openstack container set --read-acl`. A bare `--read-acl`
        is not a command, and `openstack container show` names no
        flag. The `doctl compute firewall` commands are a group.

        ```bash
        openstack container show --fenced mycontainer
        ```
      audit: |
        Run `openstack container show --long mycontainer` and read
        the ACL.
      remediation:
        - id: console
          desc: |
            Clear the ACL in Horizon, or with `swift post --read-acl ''`.
        - id: cli
          desc: |
            Clear it:

            ```bash
            swift post --read-acl '' mycontainer
            ```
  - uid: example-check-two
    title: Example two
    docs:
      desc: |-
        A chomped scalar holds `openstack user set --enable-multi-factor-auth`.
      audit: |
        Nothing to see.
"""


class TestExtractInlineCommands(unittest.TestCase):
    def setUp(self):
        self.found = extract_inline_commands(POLICY)
        self.texts = [text for text, _, _ in self.found]

    def test_extracts_commands_from_desc_and_audit_prose(self):
        self.assertIn("swift post --read-acl ''", self.texts)
        self.assertIn("openstack container set --read-acl", self.texts)
        self.assertIn("openstack container show --long mycontainer", self.texts)

    def test_skips_fenced_blocks(self):
        """extract_bash_blocks already reads those; reading them here would
        double every count."""
        self.assertNotIn("openstack container show --fenced mycontainer", self.texts)
        self.assertNotIn("swift post --read-acl '' mycontainer", self.texts)

    def test_skips_fragments_that_are_not_commands(self):
        self.assertNotIn("--read-acl", self.texts)
        self.assertNotIn("openstack container show", self.texts)
        self.assertNotIn("doctl compute firewall", self.texts)

    def test_reads_prose_of_every_remediation_entry(self):
        """A wrong command is wrong under `- id: console` too, and that entry
        is not one extract_bash_blocks scans."""
        self.assertEqual(2, self.texts.count("swift post --read-acl ''"), self.texts)

    def test_reads_chomped_block_scalars(self):
        self.assertIn("openstack user set --enable-multi-factor-auth", self.texts)

    def test_attributes_each_span_to_its_check_and_line(self):
        lines = POLICY.split("\n")
        for text, line, uid in self.found:
            self.assertIn(text.split()[0], lines[line - 1], "line number is off")
            expected = (
                "example-check-two"
                if "multi-factor" in text
                else "example-check-one"
            )
            self.assertEqual(expected, uid)

    def test_stops_at_the_end_of_the_block_scalar(self):
        """`mql:` holds backticks in other policies and is not prose."""
        self.assertNotIn("container.readAcl == null", self.texts)


class TestExtractCommandSources(unittest.TestCase):
    def test_tags_prose_spans_and_keeps_every_fenced_block(self):
        fenced = extract_bash_blocks(POLICY, include_audit=True)
        sources = extract_command_sources(POLICY, include_audit=True)

        self.assertEqual([(t, l, u) for t, l, u, prose in sources if not prose], fenced)
        self.assertEqual(
            sorted(t for t, _, _, prose in sources if prose),
            sorted(t for t, _, _ in extract_inline_commands(POLICY)),
        )


if __name__ == "__main__":
    unittest.main()
