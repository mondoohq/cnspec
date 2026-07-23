# Non-compliant: public principal "*" with no source_arn, source_account, or principal_org_id.
resource "aws_lambda_permission" "fail_example" {
  statement_id  = "AllowPublic"
  action        = "lambda:InvokeFunction"
  function_name = "my-fn"
  principal     = "*"
}
