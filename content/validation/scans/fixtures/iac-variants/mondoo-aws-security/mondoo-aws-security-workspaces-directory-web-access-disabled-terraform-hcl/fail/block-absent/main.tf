# Non-compliant: no workspace_access_properties block, so web access is not denied.
resource "aws_workspaces_directory" "fail_example" {
  directory_id = aws_directory_service_directory.example.id
  subnet_ids   = ["subnet-11111111", "subnet-22222222"]
}

resource "aws_directory_service_directory" "example" {
  name     = "corp.example.com"
  password = "SuperSecretPassw0rd"
  size     = "Small"
}
