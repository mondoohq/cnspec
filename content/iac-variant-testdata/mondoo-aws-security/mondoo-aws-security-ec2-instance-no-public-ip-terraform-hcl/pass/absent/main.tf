resource "aws_instance" "private" {
  ami           = "ami-0abcd1234"
  instance_type = "t3.micro"
  subnet_id     = "subnet-0abcd1234"
}
