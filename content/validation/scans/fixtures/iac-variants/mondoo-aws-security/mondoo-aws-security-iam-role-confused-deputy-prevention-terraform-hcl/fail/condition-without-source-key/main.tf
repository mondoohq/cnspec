# Non-compliant: the trust carries a condition, but not one that ties the
# service's request to this account or a specific resource. Covers the second
# assertion, which only looks at statements that have conditions at all.
resource "aws_iam_role" "wrong_condition" {
  name = "wrong-condition-service-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "cloudtrail.amazonaws.com" }
      Action    = "sts:AssumeRole"
      Condition = {
        Bool = { "aws:SecureTransport" = "true" }
      }
    }]
  })
}
