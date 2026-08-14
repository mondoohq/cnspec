resource "aws_appstream_fleet" "designers" {
  name          = "designers"
  instance_type = "stream.standard.medium"
  fleet_type    = "ON_DEMAND"
  image_name    = "AppStream-WinServer2019-06-17-2024"
  iam_role_arn  = aws_iam_role.appstream_streaming.arn

  compute_capacity {
    desired_instances = 2
  }
}
