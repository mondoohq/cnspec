# Compliant: a for_each'd workgroup, each instance encrypts its results.
resource "aws_athena_workgroup" "team" {
  for_each = toset(["red", "blue"])
  name     = each.key
  configuration {
    result_configuration {
      encryption_configuration {
        encryption_option = "SSE_KMS"
        kms_key_arn       = "arn:aws:kms:us-east-1:111122223333:key/abc"
      }
    }
  }
}
