# Compliant: KMS key has automatic key rotation enabled.
resource "aws_kms_key" "pass_example" {
  description         = "rotated key"
  enable_key_rotation = true
}
