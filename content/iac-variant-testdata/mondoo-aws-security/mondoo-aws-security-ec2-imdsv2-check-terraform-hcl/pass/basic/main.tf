resource "aws_instance" "pass_example" {
  ami           = "ami-0abcd1234"
  instance_type = "t3.micro"

  metadata_options {
    http_endpoint = "enabled"
    http_tokens   = "required"
  }
}
