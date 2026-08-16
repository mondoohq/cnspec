# Non-compliant: KMS key does not enable automatic key rotation.
resource "aws_kms_key" "fail_example" {
  description         = "unrotated key"
  enable_key_rotation = false
}
