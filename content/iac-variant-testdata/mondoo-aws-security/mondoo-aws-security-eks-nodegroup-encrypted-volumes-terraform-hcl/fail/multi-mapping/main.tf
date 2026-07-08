# Non-compliant: two block device mappings; one EBS volume is not encrypted.
resource "aws_launch_template" "fail_example" {
  name = "fail_example_template"

  block_device_mappings {
    device_name = "/dev/xvda"
    ebs {
      volume_size = 20
      encrypted   = true
    }
  }

  block_device_mappings {
    device_name = "/dev/xvdb"
    ebs {
      volume_size = 50
      encrypted   = false
    }
  }
}
