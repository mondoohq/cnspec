resource "aws_launch_configuration" "pass_example" {
  name                        = "pass-lc"
  image_id                    = "ami-0abcd1234"
  instance_type               = "t3.micro"
  associate_public_ip_address = false
}
