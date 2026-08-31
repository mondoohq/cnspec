# Non-compliant: role policy attachments are managed, but none attaches SecurityAudit.
resource "aws_iam_role" "audit" {
  name = "read-only"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Action    = "sts:AssumeRole"
      Principal = { AWS = "arn:aws:iam::111122223333:root" }
    }]
  })
}

resource "aws_iam_role_policy_attachment" "audit" {
  role       = aws_iam_role.audit.name
  policy_arn = "arn:aws:iam::aws:policy/ReadOnlyAccess"
}
