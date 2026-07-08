# Compliant: max user session duration is within the 36000 second limit.
resource "aws_appstream_fleet" "pass_example" {
  name          = "example-fleet"
  instance_type = "stream.standard.medium"

  compute_capacity {
    desired_instances = 1
  }

  max_user_duration_in_seconds = 36000
}
