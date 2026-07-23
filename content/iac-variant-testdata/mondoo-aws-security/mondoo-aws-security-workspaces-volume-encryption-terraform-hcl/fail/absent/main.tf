# Non-compliant: volume encryption is not configured at all.
resource "aws_workspaces_workspace" "fail_example" {
  directory_id = aws_workspaces_directory.example.id
  bundle_id    = "wsb-bh8rsxt14"
  user_name    = "jdoe"
}

resource "aws_workspaces_directory" "example" {
  directory_id = "d-1234567890"
}
