# Non-compliant: -1 is the AWS default meaning "unreserved" (no concurrency limit).
resource "aws_lambda_function" "fail_example" {
  function_name                  = "fail-fn"
  role                           = "arn:aws:iam::111122223333:role/lambda"
  handler                        = "index.handler"
  runtime                        = "nodejs18.x"
  reserved_concurrent_executions = -1
}
