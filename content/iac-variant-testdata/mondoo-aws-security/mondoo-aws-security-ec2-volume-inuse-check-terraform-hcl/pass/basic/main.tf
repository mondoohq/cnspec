resource "aws_instance" "compliant" {
  ami           = "ami-12345678"
  instance_type = "t3.micro"

  root_block_device {
    delete_on_termination = true
  }

  ebs_block_device {
    device_name           = "/dev/sdb"
    delete_on_termination = true
  }
}
