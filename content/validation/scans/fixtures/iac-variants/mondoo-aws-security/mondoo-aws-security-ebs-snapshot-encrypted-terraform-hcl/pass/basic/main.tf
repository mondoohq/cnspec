# Compliant: source EBS volume is encrypted.
resource "aws_ebs_volume" "pass_example" {
  availability_zone = "us-east-1a"
  size              = 10
  encrypted         = true
}

resource "aws_ebs_snapshot" "pass_example" {
  volume_id = aws_ebs_volume.pass_example.id
}
