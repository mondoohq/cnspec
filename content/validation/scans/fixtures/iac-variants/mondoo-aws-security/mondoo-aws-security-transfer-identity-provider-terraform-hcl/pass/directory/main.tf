# Compliant: identity provider backed by AWS Directory Service.
resource "aws_transfer_server" "ds" {
  identity_provider_type = "AWS_DIRECTORY_SERVICE"
  directory_id           = aws_directory_service_directory.example.id
}
