resource "aws_ebs_volume" "default_key" {
  availability_zone = "us-east-1a"
  size              = 100
  encrypted         = true
}
