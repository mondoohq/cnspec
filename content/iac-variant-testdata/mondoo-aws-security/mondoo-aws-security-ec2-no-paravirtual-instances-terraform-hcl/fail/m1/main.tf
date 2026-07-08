resource "aws_instance" "noncompliant" {
  ami           = "ami-12345678"
  instance_type = "m1.small"
}
