# Compliant: backup vault policy scopes access to a specific account principal.
resource "aws_backup_vault_policy" "pass_example" {
  backup_vault_name = "example-vault"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Principal = { AWS = "arn:aws:iam::111122223333:root" }
        Action    = "backup:DescribeBackupVault"
        Resource  = "*"
      }
    ]
  })
}
