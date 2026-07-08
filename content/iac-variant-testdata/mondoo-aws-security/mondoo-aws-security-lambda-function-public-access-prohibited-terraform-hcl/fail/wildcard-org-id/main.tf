# Non-compliant: public principal "*" scoped only by a wildcard principal_org_id.
resource "aws_lambda_permission" "fail_example" {
  statement_id     = "AllowPublic"
  action           = "lambda:InvokeFunction"
  function_name    = "my-fn"
  principal        = "*"
  principal_org_id = "*"
}
