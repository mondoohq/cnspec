# Non-compliant: no tracing_config block at all.
resource "aws_lambda_function" "fail_example" {
  function_name = "example"
  role          = "arn:aws:iam::123456789012:role/example"
  handler       = "index.handler"
  runtime       = "python3.12"
}
