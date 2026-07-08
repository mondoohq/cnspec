# Compliant: EBS encryption by default is enabled.
resource "aws_ebs_encryption_by_default" "pass_example" {
  enabled = true
}
