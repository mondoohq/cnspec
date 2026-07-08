# Non-compliant: one of two workgroups encrypts results with SSE_S3, not KMS.
resource "aws_athena_workgroup" "ok" {
  name = "pass-example"

  configuration {
    result_configuration {
      encryption_configuration {
        encryption_option = "SSE_KMS"
        kms_key_arn       = "arn:aws:kms:us-east-1:111122223333:key/abcd"
      }
    }
  }
}

resource "aws_athena_workgroup" "bad" {
  name = "fail-example"

  configuration {
    result_configuration {
      encryption_configuration {
        encryption_option = "SSE_S3"
      }
    }
  }
}
