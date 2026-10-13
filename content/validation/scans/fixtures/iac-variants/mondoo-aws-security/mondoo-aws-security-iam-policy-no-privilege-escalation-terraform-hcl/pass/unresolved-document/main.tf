# The policy document is built from a variable, so it does not resolve at HCL
# parse time. The check must skip this resource rather than error on it; the
# rendered plan or state is where such a policy gets evaluated.
variable "extra_action" {
  type    = string
  default = "s3:GetObject"
}

resource "aws_iam_policy" "from_variable" {
  name = "from-variable"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = [var.extra_action]
      Resource = "*"
    }]
  })
}
