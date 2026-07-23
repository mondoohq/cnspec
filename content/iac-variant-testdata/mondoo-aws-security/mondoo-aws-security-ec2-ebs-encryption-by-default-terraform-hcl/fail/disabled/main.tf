provider "aws" {
  region = "us-east-1"
}

# Non-compliant: EBS encryption by default is disabled.
resource "aws_ebs_encryption_by_default" "fail_example" {
  enabled = false
}
