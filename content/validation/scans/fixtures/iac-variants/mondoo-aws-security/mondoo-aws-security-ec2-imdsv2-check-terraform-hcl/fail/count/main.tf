# Non-compliant: counted instances leaving IMDSv2 optional.
resource "aws_instance" "counted" {
  count         = 2
  ami           = "ami-0abcd1234"
  instance_type = "t3.micro"
  metadata_options {
    http_endpoint = "enabled"
    http_tokens   = "optional"
  }
}
