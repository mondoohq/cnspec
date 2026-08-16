# Non-compliant: Lambda function does not reserve concurrent executions.
resource "aws_lambda_function" "fail_example" {
  function_name = "fail-fn"
  role          = "arn:aws:iam::111122223333:role/lambda"
  handler       = "index.handler"
  runtime       = "nodejs18.x"
}
