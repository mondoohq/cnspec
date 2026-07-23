# Non-compliant: Lambda function has no code signing config.
resource "aws_lambda_function" "fail_example" {
  function_name = "fail-fn"
  role          = "arn:aws:iam::111122223333:role/lambda"
  handler       = "index.handler"
  runtime       = "nodejs18.x"
}
