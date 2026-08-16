# Compliant: repository policy grants access to a specific principal only.
resource "aws_ecr_repository_policy" "pass_example" {
  repository = "pass-example"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowPull"
        Effect    = "Allow"
        Principal = { AWS = "arn:aws:iam::111122223333:root" }
        Action    = "ecr:GetDownloadUrlForLayer"
      }
    ]
  })
}
