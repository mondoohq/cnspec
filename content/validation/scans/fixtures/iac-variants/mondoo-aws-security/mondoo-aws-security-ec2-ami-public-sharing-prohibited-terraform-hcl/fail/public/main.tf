# Non-compliant: AMI shared publicly with all.
resource "aws_ami_launch_permission" "fail_example" {
  image_id = "ami-12345678"
  group    = "all"
}
