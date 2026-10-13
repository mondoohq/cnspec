# Compliant: read-only access, no privilege-escalation vector.
resource "aws_iam_policy" "readonly" {
  name = "readonly"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["s3:GetObject", "s3:ListBucket"]
      Resource = "arn:aws:s3:::example-bucket/*"
    }]
  })
}
