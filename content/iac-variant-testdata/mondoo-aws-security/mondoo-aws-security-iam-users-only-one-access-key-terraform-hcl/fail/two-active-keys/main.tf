# Non-compliant: the same user has two active access keys.
resource "aws_iam_access_key" "primary" {
  user = "alice"
}

resource "aws_iam_access_key" "secondary" {
  user = "alice"
}
