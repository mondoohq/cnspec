# Non-compliant: reserved concurrency explicitly set to 0 (throttles all invocations,
# but still not a positive reservation guaranteeing capacity).
resource "aws_lambda_function" "fail_example" {
  function_name                  = "fail-fn"
  role                           = "arn:aws:iam::111122223333:role/lambda"
  handler                        = "index.handler"
  runtime                        = "nodejs18.x"
  reserved_concurrent_executions = 0
}
