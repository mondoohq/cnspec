# Non-compliant: iam:CreatePolicyVersion lets the principal rewrite any managed
# policy it can reach, including one attached to itself.
resource "aws_iam_policy" "escalate" {
  name = "escalate"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["iam:CreatePolicyVersion"]
      Resource = "*"
    }]
  })
}
