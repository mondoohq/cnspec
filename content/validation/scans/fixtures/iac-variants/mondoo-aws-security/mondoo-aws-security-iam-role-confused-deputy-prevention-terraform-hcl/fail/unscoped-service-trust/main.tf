# Non-compliant: any caller able to make the service act can have it assume
# this role, because the trust carries no aws:SourceAccount or aws:SourceArn.
resource "aws_iam_role" "unscoped" {
  name = "unscoped-service-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "cloudtrail.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}
