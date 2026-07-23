# Compliant: file systems created with count, each with an enabled backup
# policy referencing the corresponding instance.
resource "aws_efs_file_system" "foo" {
  count          = 2
  creation_token = "foo-${count.index}"
}

resource "aws_efs_backup_policy" "foo" {
  count          = 2
  file_system_id = aws_efs_file_system.foo[count.index].id

  backup_policy {
    status = "ENABLED"
  }
}
