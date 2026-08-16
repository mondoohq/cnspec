# Non-compliant: backup vault policy allows every principal (Principal = "*").
resource "aws_backup_vault_policy" "fail_star" {
  backup_vault_name = "example-vault"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Principal = "*"
        Action    = "backup:DescribeBackupVault"
        Resource  = "*"
      }
    ]
  })
}
