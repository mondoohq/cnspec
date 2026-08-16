resource "aws_lambda_function" "rotation" {
  function_name = "secret-rotation"
  role          = "arn:aws:iam::123456789012:role/lambda"
  handler       = "index.handler"
  runtime       = "python3.12"
  filename      = "rotation.zip"
}
