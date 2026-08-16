# Non-compliant: idle disconnect timeout exceeds the 900 second maximum.
resource "aws_appstream_fleet" "fail_example" {
  name          = "example-fleet"
  instance_type = "stream.standard.medium"

  compute_capacity {
    desired_instances = 1
  }

  idle_disconnect_timeout_in_seconds = 3600
}
