# Compliant: query results have an encryption configuration.
resource "aws_athena_workgroup" "pass_example" {
  name = "pass-example"

  configuration {
    result_configuration {
      encryption_configuration {
        encryption_option = "SSE_S3"
      }
    }
  }
}
