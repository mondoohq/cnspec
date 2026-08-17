#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
"""Every checked-in grammar and spec in data/ has to be readable as a diff.

These files are only ever read by a machine, so nothing about their formatting
affects what the validators accept. What it affects is the one place a human is
in the loop: `validation-dependency-updates.yaml` opens a weekly pull request
refreshing them, and approving it means reading which endpoints, commands,
parameters and schemas moved.

Two of them were stored minified to save disk. A 3 MiB file on one line renders
as a single changed line -- GitHub declines to display it, `git diff` emits
megabytes of unreadable output, and reviewing the bump degrades into trusting
the vendor. The size that bought is 0.2 MiB compressed, which is what git
actually stores.

So the rule is: no dump script may emit a single-line document. This test does
not check *which* indent a file uses -- the scripts legitimately differ, some
writing indent=1 and some indent=2 -- only that the output is line-oriented
enough to diff.

    make test/content/upstream/unit
"""

import json
import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))  # content/validation
from paths import DATA_DIR  # noqa: E402

# A pretty-printed document's longest line is its longest single string value:
# 13.5 KiB today, in an Okta operation description. A minified document's
# longest line is the whole file. 64 KiB sits far above the former and far
# below the latter, so this fails only for output that is genuinely not
# line-oriented, and does not become a tripwire for a verbose vendor.
MAX_LINE_BYTES = 64 * 1024


def data_files() -> list[Path]:
    return sorted(DATA_DIR.glob("*.json"))


class DataFormatTest(unittest.TestCase):
    def test_there_are_data_files_to_check(self):
        # A wrong DATA_DIR would glob cleanly, yield nothing, and let every
        # test below pass vacuously -- the failure mode paths.py exists to
        # prevent, so it is asserted rather than assumed.
        self.assertGreater(len(data_files()), 5)

    def test_every_file_is_valid_json(self):
        for path in data_files():
            with self.subTest(path.name):
                json.loads(path.read_text())

    def test_no_file_is_a_single_line(self):
        for path in data_files():
            with self.subTest(path.name):
                lines = path.read_text().splitlines()
                self.assertGreater(
                    len(lines), 1,
                    f"{path.name} is minified onto one line, which cannot be "
                    "reviewed as a diff. Write it with json.dumps(..., indent=1) "
                    "in its upstream/dump/ script and regenerate.",
                )

    def test_no_line_is_longer_than_the_bound(self):
        for path in data_files():
            with self.subTest(path.name):
                longest = max(len(line) for line in path.read_text().splitlines())
                self.assertLessEqual(
                    longest, MAX_LINE_BYTES,
                    f"{path.name} has a {longest} byte line, over the "
                    f"{MAX_LINE_BYTES} byte bound. Either it is partially "
                    "minified, or a vendor string grew past what a reviewer "
                    "can read on one line.",
                )

    def test_every_file_ends_with_a_newline(self):
        for path in data_files():
            with self.subTest(path.name):
                self.assertTrue(
                    path.read_text().endswith("\n"),
                    f"{path.name} has no trailing newline, so the last line of "
                    "every diff against it reads as modified.",
                )


if __name__ == "__main__":
    unittest.main()
