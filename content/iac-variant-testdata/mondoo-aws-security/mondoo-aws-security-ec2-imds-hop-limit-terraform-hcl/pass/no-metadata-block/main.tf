resource "aws_instance" "no_metadata" {
  ami           = "ami-0abcd1234"
  instance_type = "t3.micro"
}
