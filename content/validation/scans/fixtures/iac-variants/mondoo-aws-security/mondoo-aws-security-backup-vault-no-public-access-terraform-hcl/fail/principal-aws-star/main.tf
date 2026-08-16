# Non-compliant: backup vault policy allows any AWS account (Principal.AWS = "*").
resource "aws_backup_vault_policy" "fail_aws_star" {
  backup_vault_name = "example-vault"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Principal = { AWS = "*" }
        Action    = "backup:DescribeBackupVault"
        Resource  = "*"
      }
    ]
  })
}
