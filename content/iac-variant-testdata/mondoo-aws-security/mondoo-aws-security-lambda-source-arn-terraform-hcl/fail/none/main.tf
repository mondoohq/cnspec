# Non-compliant: permission has neither source_arn nor source_account.
resource "aws_lambda_permission" "fail_example" {
  statement_id  = "AllowExecution"
  action        = "lambda:InvokeFunction"
  function_name = "example"
  principal     = "s3.amazonaws.com"
}
