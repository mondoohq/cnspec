resource "aws_iam_account_password_policy" "this" {
  minimum_password_length = 10
}
