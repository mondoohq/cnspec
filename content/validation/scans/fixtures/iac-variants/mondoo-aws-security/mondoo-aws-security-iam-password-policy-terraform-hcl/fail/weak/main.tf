resource "aws_iam_account_password_policy" "this" {
  minimum_password_length        = 8
  require_uppercase_characters   = false
  require_lowercase_characters   = true
  require_numbers                = true
  require_symbols                = false
  max_password_age               = 365
  password_reuse_prevention      = 2
  allow_users_to_change_password = false
}
