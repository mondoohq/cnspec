# Non-compliant: Lambda function has no vpc_config block.
resource "aws_lambda_function" "fail_example" {
  function_name = "example"
  role          = "arn:aws:iam::123456789012:role/example"
  handler       = "index.handler"
  runtime       = "python3.12"
}
