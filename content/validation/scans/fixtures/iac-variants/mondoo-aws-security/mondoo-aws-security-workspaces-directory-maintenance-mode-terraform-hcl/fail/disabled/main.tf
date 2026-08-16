# Non-compliant: maintenance mode is explicitly disabled.
resource "aws_workspaces_directory" "fail_example" {
  directory_id = aws_directory_service_directory.example.id
  subnet_ids   = ["subnet-11111111", "subnet-22222222"]

  workspace_creation_properties {
    enable_internet_access  = false
    enable_maintenance_mode = false
  }
}

resource "aws_directory_service_directory" "example" {
  name     = "corp.example.com"
  password = "SuperSecretPassw0rd"
  size     = "Small"
}
