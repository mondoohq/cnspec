resource "aws_instance" "noncompliant" {
  ami           = "ami-12345678"
  instance_type = "t3.micro"

  ebs_block_device {
    device_name           = "/dev/sdb"
    delete_on_termination = false
  }
}
