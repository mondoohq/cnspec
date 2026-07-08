# Non-compliant: encryption toggled by a conditional whose active branch is
# false.
variable "encrypt_volumes" {
  type    = bool
  default = false
}

resource "aws_launch_template" "fail_example" {
  name = "fail_example_template"

  block_device_mappings {
    device_name = "/dev/xvda"
    ebs {
      volume_size = 20
      encrypted   = var.encrypt_volumes ? true : false
    }
  }
}
