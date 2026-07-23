# Non-compliant: two file systems; exactly one lacks any backup policy. .all()
# over file systems must still fail.
resource "aws_efs_file_system" "backed" {
  creation_token = "backed"
}

resource "aws_efs_backup_policy" "backed" {
  file_system_id = aws_efs_file_system.backed.id

  backup_policy {
    status = "ENABLED"
  }
}

resource "aws_efs_file_system" "unbacked" {
  creation_token = "unbacked"
}
