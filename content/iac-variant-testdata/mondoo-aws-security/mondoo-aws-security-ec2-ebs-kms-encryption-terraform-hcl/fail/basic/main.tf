resource "aws_ebs_volume" "fail_example" {
  availability_zone = "us-east-1a"
  size              = 40
  encrypted         = false
}
