# Compliant: both root and user volumes are encrypted.
resource "aws_workspaces_workspace" "pass_example" {
  directory_id = aws_workspaces_directory.example.id
  bundle_id    = "wsb-bh8rsxt14"
  user_name    = "jdoe"

  root_volume_encryption_enabled = true
  user_volume_encryption_enabled = true
  volume_encryption_key          = "alias/aws/workspaces"
}

resource "aws_workspaces_directory" "example" {
  directory_id = "d-1234567890"
}
