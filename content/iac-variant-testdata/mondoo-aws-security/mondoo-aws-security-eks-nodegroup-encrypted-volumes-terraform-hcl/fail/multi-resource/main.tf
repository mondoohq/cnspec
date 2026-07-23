# Non-compliant: two launch templates; exactly one has an unencrypted EBS
# volume. .all() over resources must still fail.
resource "aws_launch_template" "good" {
  name = "good_template"

  block_device_mappings {
    device_name = "/dev/xvda"
    ebs {
      volume_size = 20
      encrypted   = true
    }
  }
}

resource "aws_launch_template" "bad" {
  name = "bad_template"

  block_device_mappings {
    device_name = "/dev/xvda"
    ebs {
      volume_size = 20
      encrypted   = false
    }
  }
}
