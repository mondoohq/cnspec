# Non-compliant: default internet access is enabled.
resource "aws_appstream_fleet" "fail_example" {
  name          = "example-fleet"
  instance_type = "stream.standard.medium"

  compute_capacity {
    desired_instances = 1
  }

  enable_default_internet_access = true
}
