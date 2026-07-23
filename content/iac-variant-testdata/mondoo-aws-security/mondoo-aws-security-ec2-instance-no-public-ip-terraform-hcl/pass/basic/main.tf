resource "aws_instance" "pass_example" {
  ami                         = "ami-0abcd1234"
  instance_type               = "t3.micro"
  associate_public_ip_address = false
}
