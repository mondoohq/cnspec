# Compliant: EBS volume is encrypted.
resource "aws_ebs_volume" "pass_example" {
  availability_zone = "us-east-1"
  size              = 40
  encrypted         = true
}
