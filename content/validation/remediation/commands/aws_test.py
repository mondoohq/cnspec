# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Unit tests for the AWS CLI boolean-flag check in aws.py.
#
# Run: python3 -m unittest discover -s content/validation/remediation/commands -p "*_test.py"

import unittest

from aws import AWS_GLOBAL_BOOLEAN_FLAGS, boolean_flag_value_errors

TRAIL_BOOLEANS = {"--include-shadow-trails", "--no-include-shadow-trails"} | AWS_GLOBAL_BOOLEAN_FLAGS


class BooleanFlagValueTest(unittest.TestCase):
    def test_value_after_boolean_flag_fails(self):
        # The CLI rejects this with `Unknown options: false`.
        errors = boolean_flag_value_errors(
            "aws cloudtrail describe-trails --include-shadow-trails false --query 'trailList[]'",
            TRAIL_BOOLEANS,
        )
        self.assertEqual(errors, ["boolean flag '--include-shadow-trails' takes no value, got 'false'"])

    def test_equals_value_fails(self):
        errors = boolean_flag_value_errors(
            "aws cloudtrail describe-trails --include-shadow-trails=false", TRAIL_BOOLEANS
        )
        self.assertEqual(len(errors), 1)

    def test_negated_form_passes(self):
        self.assertEqual(
            boolean_flag_value_errors(
                "aws cloudtrail describe-trails --no-include-shadow-trails --query 'trailList[]'",
                TRAIL_BOOLEANS,
            ),
            [],
        )

    def test_boolean_flag_last_or_before_pipe_passes(self):
        self.assertEqual(
            boolean_flag_value_errors("aws cloudtrail describe-trails --include-shadow-trails", TRAIL_BOOLEANS),
            [],
        )
        self.assertEqual(
            boolean_flag_value_errors(
                "aws cloudtrail describe-trails --no-paginate | jq '.trailList'", TRAIL_BOOLEANS
            ),
            [],
        )

    def test_value_flag_is_not_checked(self):
        self.assertEqual(
            boolean_flag_value_errors("aws cloudtrail describe-trails --output json", TRAIL_BOOLEANS),
            [],
        )


if __name__ == "__main__":
    unittest.main()
