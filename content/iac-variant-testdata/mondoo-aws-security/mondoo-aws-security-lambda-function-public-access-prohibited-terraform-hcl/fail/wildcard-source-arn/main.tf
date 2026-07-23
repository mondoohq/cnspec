# Non-compliant: public principal "*" scoped by a source_arn that contains a wildcard.
resource "aws_lambda_permission" "fail_example" {
  statement_id  = "AllowPublic"
  action        = "lambda:InvokeFunction"
  function_name = "my-fn"
  principal     = "*"
  source_arn    = "arn:aws:s3:::*"
}
