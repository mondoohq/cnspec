# Compliant: public principal "*" scoped by a concrete organization id.
resource "aws_lambda_permission" "pass_example" {
  statement_id     = "AllowFromOrg"
  action           = "lambda:InvokeFunction"
  function_name    = "my-fn"
  principal        = "*"
  principal_org_id = "o-abc123def4"
}
