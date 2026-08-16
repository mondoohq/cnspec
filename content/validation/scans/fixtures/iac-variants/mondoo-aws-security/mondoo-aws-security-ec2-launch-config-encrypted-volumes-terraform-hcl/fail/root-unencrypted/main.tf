resource "aws_launch_configuration" "root_unencrypted" {
  name          = "root-unenc-lc"
  image_id      = "ami-0abcd1234"
  instance_type = "t3.micro"

  root_block_device {
    volume_size = 20
    encrypted   = false
  }
}
