# Non-compliant: tracing mode is PassThrough, not Active.
resource "aws_lambda_function" "fail_example" {
  function_name = "example"
  role          = "arn:aws:iam::123456789012:role/example"
  handler       = "index.handler"
  runtime       = "python3.12"

  tracing_config {
    mode = "PassThrough"
  }
}
