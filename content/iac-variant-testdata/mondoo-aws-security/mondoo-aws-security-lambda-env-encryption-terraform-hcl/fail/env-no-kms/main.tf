# Non-compliant: environment block present but no KMS key configured.
resource "aws_lambda_function" "fail_example" {
  function_name = "fail-fn"
  role          = "arn:aws:iam::111122223333:role/lambda"
  handler       = "index.handler"
  runtime       = "nodejs18.x"

  environment {
    variables = {
      STAGE = "prod"
    }
  }
}
