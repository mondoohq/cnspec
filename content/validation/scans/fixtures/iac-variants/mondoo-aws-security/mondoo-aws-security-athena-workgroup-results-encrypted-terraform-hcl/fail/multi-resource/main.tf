# Two workgroups; the second omits the encryption configuration, so .all() must fail.
resource "aws_athena_workgroup" "compliant" {
  name = "compliant"
  configuration {
    result_configuration {
      encryption_configuration {
        encryption_option = "SSE_S3"
      }
    }
  }
}

resource "aws_athena_workgroup" "violating" {
  name = "violating"
  configuration {
    result_configuration {
      output_location = "s3://results/"
    }
  }
}
