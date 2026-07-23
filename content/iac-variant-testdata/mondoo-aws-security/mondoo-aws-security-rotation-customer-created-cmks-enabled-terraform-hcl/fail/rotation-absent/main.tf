# Non-compliant: KMS key omits enable_key_rotation, so automatic rotation is off.
resource "aws_kms_key" "fail_example" {
  description = "key with no rotation configured"
}
