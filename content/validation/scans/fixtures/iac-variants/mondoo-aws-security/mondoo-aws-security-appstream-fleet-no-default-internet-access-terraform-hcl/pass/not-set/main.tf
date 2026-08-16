# Compliant: default internet access is not set, so it defaults to disabled.
resource "aws_appstream_fleet" "pass_example" {
  name          = "example-fleet"
  instance_type = "stream.standard.medium"

  compute_capacity {
    desired_instances = 1
  }
}
