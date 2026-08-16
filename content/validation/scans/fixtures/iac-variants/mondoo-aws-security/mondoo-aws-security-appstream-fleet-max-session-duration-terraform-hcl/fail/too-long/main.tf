# Non-compliant: max user session duration exceeds the 36000 second limit.
resource "aws_appstream_fleet" "fail_example" {
  name          = "example-fleet"
  instance_type = "stream.standard.medium"

  compute_capacity {
    desired_instances = 1
  }

  max_user_duration_in_seconds = 57600
}
