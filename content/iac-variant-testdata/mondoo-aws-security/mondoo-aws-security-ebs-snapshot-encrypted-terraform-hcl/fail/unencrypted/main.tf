# Non-compliant: source EBS volume is not encrypted.
resource "aws_ebs_volume" "fail_example" {
  availability_zone = "us-east-1a"
  size              = 10
  encrypted         = false
}

resource "aws_ebs_snapshot" "fail_example" {
  volume_id = aws_ebs_volume.fail_example.id
}
