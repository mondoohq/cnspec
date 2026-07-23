# Compliant: default internet access is disabled.
resource "aws_appstream_fleet" "pass_example" {
  name          = "example-fleet"
  instance_type = "stream.standard.medium"

  compute_capacity {
    desired_instances = 1
  }

  enable_default_internet_access = false
}
