# Two instances; the second allows an oversized IMDS hop limit.
resource "aws_instance" "compliant" {
  ami           = "ami-0abcd1234"
  instance_type = "t3.micro"
  metadata_options {
    http_tokens                 = "required"
    http_put_response_hop_limit = 1
  }
}

resource "aws_instance" "violating" {
  ami           = "ami-0abcd1234"
  instance_type = "t3.micro"
  metadata_options {
    http_tokens                 = "required"
    http_put_response_hop_limit = 3
  }
}
