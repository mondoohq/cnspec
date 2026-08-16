# Compliant: directory enables WorkSpaces maintenance mode.
resource "aws_workspaces_directory" "pass_example" {
  directory_id = aws_directory_service_directory.example.id
  subnet_ids   = ["subnet-11111111", "subnet-22222222"]

  workspace_creation_properties {
    enable_internet_access  = false
    enable_maintenance_mode = true
    user_enabled_as_local_administrator = false
  }
}

resource "aws_directory_service_directory" "example" {
  name     = "corp.example.com"
  password = "SuperSecretPassw0rd"
  size     = "Small"
}
