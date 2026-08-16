schema_version = 1

project {
  license          = "BUSL-1.1"
  copyright_holder = "Mondoo, Inc."
  copyright_year   = 2024

  # (OPTIONAL) A list of globs that should not have copyright/license headers.
  # Supports doublestar glob patterns for more flexibility in defining which
  # files or folders should be ignored
  header_ignore = [
    "**/*.tf",
    "**/testdata/**",
    # Content validation fixtures: captured IaC input, not source we author.
    # Was **/iac-variant-testdata/** before the fixtures moved under validation/.
    "content/validation/scans/fixtures/**",
    "**/*.pb.go",
    "**/*_string.go",
    "apps/cnspec/cmd/policy-example.mql.yaml",
  ]
}