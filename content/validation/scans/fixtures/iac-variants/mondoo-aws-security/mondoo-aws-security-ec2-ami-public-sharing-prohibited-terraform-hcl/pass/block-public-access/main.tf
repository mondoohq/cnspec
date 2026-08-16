resource "aws_ami" "example" {
  name                = "example-ami"
  virtualization_type = "hvm"
  root_device_name    = "/dev/xvda"

  ebs_block_device {
    device_name = "/dev/xvda"
    snapshot_id = "snap-0123456789abcdef0"
  }
}

resource "aws_ec2_image_block_public_access" "this" {
  state = "block-new-sharing"
}
