# Non-compliant: public principal "*" scoped only by a wildcard source_account.
resource "aws_lambda_permission" "fail_example" {
  statement_id   = "AllowPublic"
  action         = "lambda:InvokeFunction"
  function_name  = "my-fn"
  principal      = "*"
  source_account = "*"
}
