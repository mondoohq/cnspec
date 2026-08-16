# Non-compliant: function URL allows unauthenticated (NONE) access.
resource "aws_lambda_function_url" "fail_example" {
  function_name      = "example"
  authorization_type = "NONE"
}
