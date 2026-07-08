# Non-compliant: ebs block present but encrypted omitted, defaulting to unencrypted.
resource "aws_launch_template" "fail_example" {
  name = "fail_example_template"

  block_device_mappings {
    device_name = "/dev/xvda"
    ebs {
      volume_size = 20
    }
  }
}
