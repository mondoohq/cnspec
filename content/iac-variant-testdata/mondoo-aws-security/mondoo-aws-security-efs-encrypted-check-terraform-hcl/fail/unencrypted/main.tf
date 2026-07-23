resource "aws_efs_file_system" "fail" {
  creation_token = "fail"
  encrypted      = false
}
