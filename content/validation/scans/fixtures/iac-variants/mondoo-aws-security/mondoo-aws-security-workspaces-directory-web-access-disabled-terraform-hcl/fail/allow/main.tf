# Non-compliant: web access to WorkSpaces is allowed.
resource "aws_workspaces_directory" "fail_example" {
  directory_id = aws_directory_service_directory.example.id
  subnet_ids   = ["subnet-11111111", "subnet-22222222"]

  workspace_access_properties {
    device_type_web     = "ALLOW"
    device_type_windows = "ALLOW"
  }
}

resource "aws_directory_service_directory" "example" {
  name     = "corp.example.com"
  password = "SuperSecretPassw0rd"
  size     = "Small"
}
