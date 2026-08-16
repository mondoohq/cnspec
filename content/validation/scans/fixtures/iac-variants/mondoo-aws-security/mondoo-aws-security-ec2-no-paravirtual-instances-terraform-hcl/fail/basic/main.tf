resource "aws_instance" "noncompliant" {
  ami           = "ami-12345678"
  instance_type = "t1.micro"
}
