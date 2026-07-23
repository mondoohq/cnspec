# Non-compliant: user volume is encrypted but the root volume is not.
resource "aws_workspaces_workspace" "fail_example" {
  directory_id = aws_workspaces_directory.example.id
  bundle_id    = "wsb-bh8rsxt14"
  user_name    = "jdoe"

  root_volume_encryption_enabled = false
  user_volume_encryption_enabled = true
}

resource "aws_workspaces_directory" "example" {
  directory_id = "d-1234567890"
}
