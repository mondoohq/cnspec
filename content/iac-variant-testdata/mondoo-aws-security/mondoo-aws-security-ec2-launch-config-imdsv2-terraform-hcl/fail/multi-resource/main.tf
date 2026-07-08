# Two launch configurations; the second leaves IMDSv2 optional.
resource "aws_launch_configuration" "compliant" {
  name          = "compliant-lc"
  image_id      = "ami-0abcd1234"
  instance_type = "t3.micro"
  metadata_options {
    http_tokens = "required"
  }
}

resource "aws_launch_configuration" "violating" {
  name          = "violating-lc"
  image_id      = "ami-0abcd1234"
  instance_type = "t3.micro"
  metadata_options {
    http_tokens = "optional"
  }
}
