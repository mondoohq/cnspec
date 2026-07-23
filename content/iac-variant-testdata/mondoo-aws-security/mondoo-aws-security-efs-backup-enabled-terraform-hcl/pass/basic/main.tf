resource "aws_efs_file_system" "foo" {
  creation_token = "foo"
}

resource "aws_efs_backup_policy" "foo" {
  file_system_id = aws_efs_file_system.foo.id

  backup_policy {
    status = "ENABLED"
  }
}
