resource "aws_instance" "imds_disabled" {
  ami           = "ami-0abcd1234"
  instance_type = "t3.micro"

  metadata_options {
    http_endpoint = "disabled"
  }
}
