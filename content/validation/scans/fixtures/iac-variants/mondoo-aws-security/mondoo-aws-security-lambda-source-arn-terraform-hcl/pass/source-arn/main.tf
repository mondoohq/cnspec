# Compliant: permission scoped by source_arn.
resource "aws_lambda_permission" "pass_example" {
  statement_id  = "AllowExecution"
  action        = "lambda:InvokeFunction"
  function_name = "example"
  principal     = "s3.amazonaws.com"
  source_arn    = "arn:aws:s3:::example-bucket"
}
