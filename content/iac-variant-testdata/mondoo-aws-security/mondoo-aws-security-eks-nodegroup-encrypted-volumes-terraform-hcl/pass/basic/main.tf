# Compliant: launch template EBS volume is encrypted.
resource "aws_launch_template" "pass_example" {
  name = "pass_example_template"

  block_device_mappings {
    device_name = "/dev/xvda"
    ebs {
      volume_size = 20
      encrypted   = true
    }
  }
}
