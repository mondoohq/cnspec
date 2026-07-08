resource "aws_launch_configuration" "no_block_devices" {
  name          = "no-bd-lc"
  image_id      = "ami-0abcd1234"
  instance_type = "t3.micro"
}
