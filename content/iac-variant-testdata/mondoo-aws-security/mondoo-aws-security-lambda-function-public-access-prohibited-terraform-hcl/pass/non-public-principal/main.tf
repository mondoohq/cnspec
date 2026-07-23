# Compliant: invocation granted to a specific AWS service, not the public "*".
resource "aws_lambda_permission" "pass_example" {
  statement_id  = "AllowS3Invoke"
  action        = "lambda:InvokeFunction"
  function_name = "my-fn"
  principal     = "s3.amazonaws.com"
  source_arn    = "arn:aws:s3:::my-specific-bucket"
}
