# Without minimum encryption enforcement a query author can override the
# workgroup's encryption settings and write unencrypted results.
resource "aws_athena_workgroup" "analytics" {
  name = "analytics"

  configuration {
    enforce_workgroup_configuration         = true
    enable_minimum_encryption_configuration = false

    result_configuration {
      output_location = "s3://example-athena-results/analytics/"
    }
  }
}
