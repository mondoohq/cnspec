# Compliant: Lambda function has a code signing config attached.
resource "aws_lambda_function" "pass_example" {
  function_name         = "pass-fn"
  role                  = "arn:aws:iam::111122223333:role/lambda"
  handler               = "index.handler"
  runtime               = "nodejs18.x"
  code_signing_config_arn = "arn:aws:lambda:us-east-1:111122223333:code-signing-config:csc-0123"
}
