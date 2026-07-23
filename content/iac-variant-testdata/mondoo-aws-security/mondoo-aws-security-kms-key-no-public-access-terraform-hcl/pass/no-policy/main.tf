# Compliant: KMS key with no inline policy (arguments.policy == empty).
resource "aws_kms_key" "pass_example" {
  description = "no policy key"
}
