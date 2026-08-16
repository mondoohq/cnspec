resource "aws_launch_template" "compliant" {
  name          = "example"
  image_id      = "ami-12345678"
  instance_type = "t3.micro"
  user_data     = "echo hello world"
}
