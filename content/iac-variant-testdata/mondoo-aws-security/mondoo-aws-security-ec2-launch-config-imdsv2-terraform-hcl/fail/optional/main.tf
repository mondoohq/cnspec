resource "aws_launch_configuration" "fail_example" {
  name          = "fail-lc"
  image_id      = "ami-0abcd1234"
  instance_type = "t3.micro"

  metadata_options {
    http_tokens = "optional"
  }
}
