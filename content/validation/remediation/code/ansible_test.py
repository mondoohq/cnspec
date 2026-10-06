#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1

import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import unittest
from unittest import mock

spec = importlib.util.spec_from_file_location(
    "ansible_remediation_validator", Path(__file__).with_name("ansible.py")
)
validator = importlib.util.module_from_spec(spec)
sys.modules[spec.name] = validator
spec.loader.exec_module(validator)


class TestUnknownModuleResults(unittest.TestCase):
    def lint_result(self, description: str):
        output = json.dumps(
            [
                {
                    "check_name": "syntax-check[unknown-module]",
                    "description": description,
                }
            ]
        )
        completed = subprocess.CompletedProcess(
            args=["ansible-lint"], returncode=2, stdout=output, stderr=""
        )
        with mock.patch.object(validator.subprocess, "run", return_value=completed):
            return validator.run_ansible_lint(Path("playbook.yml"))

    def test_missing_builtin_module_fails(self):
        result = self.lint_result(
            "couldn't resolve module/action 'ansible.builtin.ini_file'. "
            "This often indicates a misspelling, missing collection, or incorrect module path."
        )
        self.assertFalse(result.success)
        self.assertIn("ansible.builtin.ini_file", result.issues[0])

    def test_missing_collection_module_is_ignored_offline(self):
        result = self.lint_result(
            "couldn't resolve module/action 'community.general.ini_file'. "
            "This often indicates a misspelling, missing collection, or incorrect module path."
        )
        self.assertTrue(result.success)
        self.assertEqual([], result.issues)

    def test_unknown_module_without_name_fails_closed(self):
        result = self.lint_result("couldn't resolve module/action")
        self.assertFalse(result.success)
        self.assertIn("syntax-check[unknown-module]", result.issues[0])

    def test_redirected_builtin_names_fail_even_when_lint_would_pass(self):
        block = validator.AnsibleBlock(
            code="""- hosts: all
  tasks:
    - ansible.builtin.sysctl:
        name: fs.suid_dumpable
        value: '0'
    - ansible.builtin.zypper:
        name: prelink
        state: absent
""",
            line=1,
            uid="test-redirects",
            file=Path("policy.yml"),
        )
        with mock.patch.object(validator, "run_ansible_lint") as lint:
            _, success, issues = validator.validate_block(
                block, {"ansible.builtin.command"}
            )
        self.assertFalse(success)
        self.assertEqual(
            [
                "Unknown Ansible built-in module: ansible.builtin.sysctl",
                "Unknown Ansible built-in module: ansible.builtin.zypper",
            ],
            issues,
        )
        lint.assert_not_called()

    def test_builtin_inventory_failure_does_not_pass_validation(self):
        completed = subprocess.CompletedProcess(
            args=["ansible-doc"], returncode=0, stdout="", stderr=""
        )
        with mock.patch.object(validator.subprocess, "run", return_value=completed):
            with self.assertRaisesRegex(RuntimeError, "Could not list"):
                validator.load_builtin_modules()


if __name__ == "__main__":
    unittest.main()
