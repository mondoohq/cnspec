# Non-compliant: inline policies attached via a dynamic "inline_policy" block.
# A user with an inline policy must be flagged regardless of how the block is authored.
variable "inline_policies" {
  type = map(string)
  default = {
    s3 = "s3:GetObject"
  }
}

resource "aws_iam_user" "u" {
  name = "app-user"

  dynamic "inline_policy" {
    for_each = var.inline_policies
    content {
      name   = inline_policy.key
      policy = jsonencode({ Version = "2012-10-17", Statement = [{ Effect = "Allow", Action = inline_policy.value, Resource = "*" }] })
    }
  }
}
