resource "aws_instance" "compliant" {
  ami           = "ami-12345678"
  instance_type = "t3.micro"

  root_block_device {
    volume_size = 50
  }
}
