# Non-compliant: a counted launch configuration with an unencrypted EBS volume.
resource "aws_launch_configuration" "counted" {
  count         = 2
  name          = "counted-lc-${count.index}"
  image_id      = "ami-0abcd1234"
  instance_type = "t3.micro"

  root_block_device {
    encrypted = true
  }

  ebs_block_device {
    device_name = "/dev/sdb"
    encrypted   = false
  }
}
