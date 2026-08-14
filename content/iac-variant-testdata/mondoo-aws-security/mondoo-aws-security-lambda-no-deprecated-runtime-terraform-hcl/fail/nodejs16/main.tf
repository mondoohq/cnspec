resource "aws_lambda_function" "webhook" {
  function_name = "webhook-handler"
  role          = aws_iam_role.lambda.arn
  handler       = "index.handler"
  runtime       = "nodejs16.x"
  filename      = "function.zip"
}
