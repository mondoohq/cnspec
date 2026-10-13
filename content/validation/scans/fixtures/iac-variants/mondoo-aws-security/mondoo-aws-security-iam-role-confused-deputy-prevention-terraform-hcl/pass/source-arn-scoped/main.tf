# Compliant: the service trust is scoped with aws:SourceAccount, so another
# customer's resource cannot make the service assume this role on their behalf.
resource "aws_iam_role" "scoped" {
  name = "scoped-service-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "cloudtrail.amazonaws.com" }
      Action    = "sts:AssumeRole"
      Condition = {
        StringEquals = { "aws:SourceAccount" = "123456789012" }
      }
    }]
  })
}
