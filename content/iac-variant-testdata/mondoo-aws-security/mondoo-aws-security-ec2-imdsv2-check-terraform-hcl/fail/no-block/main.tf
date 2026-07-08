resource "aws_instance" "fail_example" {
  ami           = "ami-0abcd1234"
  instance_type = "t3.micro"
}
