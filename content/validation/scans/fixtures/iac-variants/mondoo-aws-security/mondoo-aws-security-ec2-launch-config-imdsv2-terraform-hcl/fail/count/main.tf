# Non-compliant: counted launch configurations leaving IMDSv2 optional.
resource "aws_launch_configuration" "counted" {
  count         = 2
  name          = "counted-lc-${count.index}"
  image_id      = "ami-0abcd1234"
  instance_type = "t3.micro"
  metadata_options {
    http_tokens = "optional"
  }
}
