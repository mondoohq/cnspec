# Non-compliant: counted launch templates leaving IMDSv2 optional.
resource "aws_launch_template" "counted" {
  count         = 2
  name          = "counted-lt-${count.index}"
  image_id      = "ami-0abcd1234"
  instance_type = "t3.micro"
  metadata_options {
    http_tokens = "optional"
  }
}
