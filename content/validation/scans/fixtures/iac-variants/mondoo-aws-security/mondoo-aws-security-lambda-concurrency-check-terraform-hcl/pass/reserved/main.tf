# Compliant: Lambda function reserves concurrent executions (> 0).
resource "aws_lambda_function" "pass_example" {
  function_name                  = "pass-fn"
  role                           = "arn:aws:iam::111122223333:role/lambda"
  handler                        = "index.handler"
  runtime                        = "nodejs18.x"
  reserved_concurrent_executions = 10
}
