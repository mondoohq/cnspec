# Non-compliant: no logging_role, so connector activity is not logged.
resource "aws_transfer_connector" "unlogged" {
  access_role = aws_iam_role.access.arn
  url         = "https://sftp.partner.example.com"

  sftp_config {
    trusted_host_keys = ["ssh-rsa AAAAB3Nza..."]
    user_secret_id    = aws_secretsmanager_secret.sftp.id
  }
}
