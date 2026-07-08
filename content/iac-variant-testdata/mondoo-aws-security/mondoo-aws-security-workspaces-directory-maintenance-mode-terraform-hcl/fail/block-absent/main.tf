# Non-compliant: no workspace_creation_properties block, so maintenance mode is unmanaged.
resource "aws_workspaces_directory" "fail_example" {
  directory_id = aws_directory_service_directory.example.id
  subnet_ids   = ["subnet-11111111", "subnet-22222222"]
}

resource "aws_directory_service_directory" "example" {
  name     = "corp.example.com"
  password = "SuperSecretPassw0rd"
  size     = "Small"
}
