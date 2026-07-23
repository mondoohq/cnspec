# Two instances; the second leaves IMDSv2 optional.
resource "aws_instance" "compliant" {
  ami           = "ami-0abcd1234"
  instance_type = "t3.micro"
  metadata_options {
    http_endpoint = "enabled"
    http_tokens   = "required"
  }
}

resource "aws_instance" "violating" {
  ami           = "ami-0abcd1234"
  instance_type = "t3.micro"
  metadata_options {
    http_endpoint = "enabled"
    http_tokens   = "optional"
  }
}
