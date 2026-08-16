resource "aws_instance" "compliant" {
  ami           = "ami-12345678"
  instance_type = "t3.micro"
}
