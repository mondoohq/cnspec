resource "aws_transfer_server" "example" {
  endpoint_type          = "PUBLIC"
  identity_provider_type = "SERVICE_MANAGED"
  logging_role           = aws_iam_role.transfer_logging.arn

  tags = {
    Name = "sftp-server"
  }
}
