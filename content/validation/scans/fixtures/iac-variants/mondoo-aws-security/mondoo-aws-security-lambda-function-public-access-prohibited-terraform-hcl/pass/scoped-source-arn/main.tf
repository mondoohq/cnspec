# Compliant: public principal "*" is scoped by a concrete source_arn with no wildcard.
resource "aws_lambda_permission" "pass_example" {
  statement_id  = "AllowExecution"
  action        = "lambda:InvokeFunction"
  function_name = "my-fn"
  principal     = "*"
  source_arn    = "arn:aws:s3:::my-specific-bucket"
}
