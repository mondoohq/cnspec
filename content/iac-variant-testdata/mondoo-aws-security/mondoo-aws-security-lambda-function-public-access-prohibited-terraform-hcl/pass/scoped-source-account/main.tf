# Compliant: public principal "*" scoped by a concrete AWS account id.
resource "aws_lambda_permission" "pass_example" {
  statement_id   = "AllowFromAccount"
  action         = "lambda:InvokeFunction"
  function_name  = "my-fn"
  principal      = "*"
  source_account = "111122223333"
}
