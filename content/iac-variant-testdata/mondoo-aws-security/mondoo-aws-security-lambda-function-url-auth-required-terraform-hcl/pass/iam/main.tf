# Compliant: function URL requires IAM authorization.
resource "aws_lambda_function_url" "pass_example" {
  function_name      = "example"
  authorization_type = "AWS_IAM"
}
