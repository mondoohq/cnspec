resource "aws_instance" "noncompliant" {
  ami           = "ami-12345678"
  instance_type = "t3.micro"
  user_data     = "export AWS_ACCESS_KEY_ID=AKIAABCDEFGHIJKLMNOP"
}
