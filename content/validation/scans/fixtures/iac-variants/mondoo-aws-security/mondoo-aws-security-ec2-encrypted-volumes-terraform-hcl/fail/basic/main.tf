# Non-compliant: EBS volume is not encrypted.
resource "aws_ebs_volume" "fail_example" {
  availability_zone = "us-east-1"
  size              = 40
  encrypted         = false
}
