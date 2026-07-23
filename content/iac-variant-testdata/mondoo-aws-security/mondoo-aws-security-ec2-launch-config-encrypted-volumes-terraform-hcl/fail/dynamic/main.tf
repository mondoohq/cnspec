# Non-compliant: unencrypted extra EBS volumes declared with a dynamic block.
# Modules commonly iterate additional data volumes this way.
variable "data_volumes" {
  type    = list(object({ device_name = string }))
  default = [{ device_name = "/dev/sdb" }, { device_name = "/dev/sdc" }]
}

resource "aws_launch_configuration" "fail_dynamic" {
  name          = "fail-dynamic-lc"
  image_id      = "ami-0abcd1234"
  instance_type = "t3.micro"

  root_block_device {
    encrypted = true
  }

  dynamic "ebs_block_device" {
    for_each = var.data_volumes
    content {
      device_name = ebs_block_device.value.device_name
      encrypted   = false
    }
  }
}
