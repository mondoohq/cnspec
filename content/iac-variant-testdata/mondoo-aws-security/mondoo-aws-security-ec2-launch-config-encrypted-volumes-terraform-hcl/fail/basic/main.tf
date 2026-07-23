resource "aws_launch_configuration" "fail_example" {
  name          = "fail-lc"
  image_id      = "ami-0abcd1234"
  instance_type = "t3.micro"

  root_block_device {
    encrypted = true
  }

  ebs_block_device {
    device_name = "/dev/sdb"
    encrypted   = false
  }
}
