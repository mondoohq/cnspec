# Non-compliant: the source EBS volume omits the encrypted argument, which
# defaults to unencrypted.
resource "aws_ebs_volume" "fail_example" {
  availability_zone = "us-east-1a"
  size              = 10
}

resource "aws_ebs_snapshot" "fail_example" {
  volume_id = aws_ebs_volume.fail_example.id
}
