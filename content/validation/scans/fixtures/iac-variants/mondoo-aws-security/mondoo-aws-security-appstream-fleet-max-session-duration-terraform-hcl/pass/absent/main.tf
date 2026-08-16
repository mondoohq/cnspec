# Compliant: max_user_duration_in_seconds omitted -> AWS default 960s (< 36000).
resource "aws_appstream_fleet" "pass" {
  name          = "example"
  instance_type = "stream.standard.medium"
  compute_capacity {
    desired_instances = 1
  }
}
