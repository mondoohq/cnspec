# Two launch configurations; the second leaves its EBS volume unencrypted.
resource "aws_launch_configuration" "compliant" {
  name          = "compliant-lc"
  image_id      = "ami-0abcd1234"
  instance_type = "t3.micro"

  root_block_device {
    encrypted = true
  }

  ebs_block_device {
    device_name = "/dev/sdb"
    encrypted   = true
  }
}

resource "aws_launch_configuration" "violating" {
  name          = "violating-lc"
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
