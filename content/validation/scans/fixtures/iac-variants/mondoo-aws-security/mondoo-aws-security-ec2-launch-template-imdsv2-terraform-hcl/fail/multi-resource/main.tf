# Two launch templates; the second leaves IMDSv2 optional.
resource "aws_launch_template" "compliant" {
  name          = "compliant-lt"
  image_id      = "ami-0abcd1234"
  instance_type = "t3.micro"
  metadata_options {
    http_tokens = "required"
  }
}

resource "aws_launch_template" "violating" {
  name          = "violating-lt"
  image_id      = "ami-0abcd1234"
  instance_type = "t3.micro"
  metadata_options {
    http_tokens = "optional"
  }
}
