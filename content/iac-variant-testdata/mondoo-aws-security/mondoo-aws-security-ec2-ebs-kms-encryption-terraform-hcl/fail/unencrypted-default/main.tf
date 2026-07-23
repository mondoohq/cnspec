resource "aws_ebs_volume" "no_encryption" {
  availability_zone = "us-east-1b"
  size              = 50

  tags = {
    Name = "data-volume"
  }
}
