# Non-compliant: no idle disconnect timeout set, so idle sessions are never
# disconnected (AWS default is 0 / disabled).
resource "aws_appstream_fleet" "fail_example" {
  name          = "example-fleet"
  instance_type = "stream.standard.medium"

  compute_capacity {
    desired_instances = 1
  }
}
