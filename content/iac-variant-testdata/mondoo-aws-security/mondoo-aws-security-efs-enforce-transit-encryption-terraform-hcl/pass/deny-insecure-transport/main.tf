resource "aws_efs_file_system" "shared" {
  creation_token = "shared-app-data"
  encrypted      = true
}

resource "aws_efs_file_system_policy" "shared" {
  file_system_id = aws_efs_file_system.shared.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "DenyInsecureTransport"
        Effect = "Deny"
        Principal = {
          AWS = "*"
        }
        Action   = "*"
        Resource = "arn:aws:elasticfilesystem:us-east-1:123456789012:file-system/fs-0123456789abcdef0"
        Condition = {
          Bool = {
            "aws:SecureTransport" = "false"
          }
        }
      },
    ]
  })
}
