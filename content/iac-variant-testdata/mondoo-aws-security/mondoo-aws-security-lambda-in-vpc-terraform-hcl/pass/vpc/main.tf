# Compliant: Lambda function is attached to a VPC.
resource "aws_lambda_function" "pass_example" {
  function_name = "example"
  role          = "arn:aws:iam::123456789012:role/example"
  handler       = "index.handler"
  runtime       = "python3.12"

  vpc_config {
    subnet_ids         = ["subnet-12345678"]
    security_group_ids = ["sg-12345678"]
  }
}
