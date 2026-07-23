# Compliant: web access to WorkSpaces is denied.
resource "aws_workspaces_directory" "pass_example" {
  directory_id = aws_directory_service_directory.example.id
  subnet_ids   = ["subnet-11111111", "subnet-22222222"]

  workspace_access_properties {
    device_type_web     = "DENY"
    device_type_windows = "ALLOW"
    device_type_osx     = "ALLOW"
  }
}

resource "aws_directory_service_directory" "example" {
  name     = "corp.example.com"
  password = "SuperSecretPassw0rd"
  size     = "Small"
}
