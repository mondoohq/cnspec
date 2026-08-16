# A file system policy exists, but it only grants access; nothing denies
# non-TLS mounts.
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
        Sid    = "AllowAppRole"
        Effect = "Allow"
        Principal = {
          AWS = "arn:aws:iam::123456789012:role/app"
        }
        Action = [
          "elasticfilesystem:ClientMount",
          "elasticfilesystem:ClientWrite",
        ]
        Resource = "arn:aws:elasticfilesystem:us-east-1:123456789012:file-system/fs-0123456789abcdef0"
      },
    ]
  })
}
