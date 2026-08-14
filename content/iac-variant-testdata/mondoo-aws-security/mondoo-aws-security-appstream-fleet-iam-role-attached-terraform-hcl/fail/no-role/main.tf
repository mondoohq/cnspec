# With no fleet role, streaming instances cannot be granted scoped AWS access,
# so credentials end up baked into the image or the session instead.
resource "aws_appstream_fleet" "designers" {
  name          = "designers"
  instance_type = "stream.standard.medium"
  fleet_type    = "ON_DEMAND"
  image_name    = "AppStream-WinServer2019-06-17-2024"

  compute_capacity {
    desired_instances = 2
  }
}
