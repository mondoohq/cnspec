resource "aws_instance" "noncompliant" {
  ami           = "ami-12345678"
  instance_type = "t3.micro"

  root_block_device {
    delete_on_termination = false
  }
}
