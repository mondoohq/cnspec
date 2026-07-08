resource "aws_instance" "noncompliant" {
  ami               = "ami-12345678"
  instance_type     = "t3.micro"
  source_dest_check = false
}
