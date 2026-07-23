# Non-compliant: inline policy attached via the aws_iam_user inline_policy block.
resource "aws_iam_user" "u" {
  name = "app-user"
  inline_policy {
    name   = "inline"
    policy = jsonencode({ Version = "2012-10-17", Statement = [{ Effect = "Allow", Action = "*", Resource = "*" }] })
  }
}
