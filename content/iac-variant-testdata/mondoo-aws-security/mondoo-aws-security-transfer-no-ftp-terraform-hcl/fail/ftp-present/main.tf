resource "aws_transfer_server" "example" {
  endpoint_type          = "VPC"
  identity_provider_type = "API_GATEWAY"
  protocols              = ["SFTP", "FTP"]
  url                    = "https://example.com/auth"
}
