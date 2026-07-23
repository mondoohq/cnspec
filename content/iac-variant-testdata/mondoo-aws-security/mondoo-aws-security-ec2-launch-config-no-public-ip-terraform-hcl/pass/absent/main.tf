resource "aws_launch_configuration" "private_lc" {
  name          = "private-lc"
  image_id      = "ami-0abcd1234"
  instance_type = "t3.micro"
}
