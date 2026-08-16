# Compliant: idle disconnect timeout is within 1-900 seconds.
resource "aws_appstream_fleet" "pass_example" {
  name          = "example-fleet"
  instance_type = "stream.standard.medium"

  compute_capacity {
    desired_instances = 1
  }

  idle_disconnect_timeout_in_seconds = 900
}
