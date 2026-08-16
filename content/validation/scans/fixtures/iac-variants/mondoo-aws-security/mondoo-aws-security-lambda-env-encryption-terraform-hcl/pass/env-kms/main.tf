# Compliant: environment block present and encrypted with a customer KMS key.
resource "aws_lambda_function" "pass_example" {
  function_name = "pass-fn"
  role          = "arn:aws:iam::111122223333:role/lambda"
  handler       = "index.handler"
  runtime       = "nodejs18.x"
  kms_key_arn   = "arn:aws:kms:us-east-1:111122223333:key/abcd"

  environment {
    variables = {
      STAGE = "prod"
    }
  }
}
