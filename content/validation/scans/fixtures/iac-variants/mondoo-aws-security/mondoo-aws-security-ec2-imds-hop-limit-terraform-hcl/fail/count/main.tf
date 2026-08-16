# Non-compliant: counted instances with an oversized IMDS hop limit.
resource "aws_instance" "counted" {
  count         = 2
  ami           = "ami-0abcd1234"
  instance_type = "t3.micro"
  metadata_options {
    http_tokens                 = "required"
    http_put_response_hop_limit = 3
  }
}
