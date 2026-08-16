# Compliant: transfer connector has a CloudWatch logging role.
resource "aws_transfer_connector" "logged" {
  access_role  = aws_iam_role.access.arn
  logging_role = aws_iam_role.logging.arn
  url          = "https://sftp.partner.example.com"

  sftp_config {
    trusted_host_keys = ["ssh-rsa AAAAB3Nza..."]
    user_secret_id    = aws_secretsmanager_secret.sftp.id
  }
}
