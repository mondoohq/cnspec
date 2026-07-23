# Non-compliant: AMI shared with a specific account.
resource "aws_ami_launch_permission" "fail_example" {
  image_id   = "ami-12345678"
  account_id = "111122223333"
}
