resource "aws_efs_file_system_policy" "fail" {
  file_system_id = aws_efs_file_system.this.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Principal = "*"
        Action    = ["elasticfilesystem:ClientMount"]
        Resource  = aws_efs_file_system.this.arn
      }
    ]
  })
}

resource "aws_efs_file_system" "this" {
  creation_token = "app-data"
}
