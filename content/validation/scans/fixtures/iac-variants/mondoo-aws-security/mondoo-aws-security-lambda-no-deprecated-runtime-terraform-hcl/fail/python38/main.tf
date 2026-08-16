# python3.8 is past end of support: no security patches reach the runtime.
resource "aws_lambda_function" "processor" {
  function_name = "event-processor"
  role          = aws_iam_role.lambda.arn
  handler       = "index.handler"
  runtime       = "python3.8"
  filename      = "function.zip"
}
