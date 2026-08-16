# Non-compliant: result configuration has no encryption configuration.
resource "aws_athena_workgroup" "fail_example" {
  name = "fail-example"

  configuration {
    result_configuration {
      output_location = "s3://example-bucket/output/"
    }
  }
}
