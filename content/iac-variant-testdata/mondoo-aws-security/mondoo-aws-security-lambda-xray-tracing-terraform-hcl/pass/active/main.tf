# Compliant: active X-Ray tracing enabled.
resource "aws_lambda_function" "pass_example" {
  function_name = "example"
  role          = "arn:aws:iam::123456789012:role/example"
  handler       = "index.handler"
  runtime       = "python3.12"

  tracing_config {
    mode = "Active"
  }
}
