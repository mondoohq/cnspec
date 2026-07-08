# Non-compliant: method has no authorization and does not require an API key.
resource "aws_api_gateway_method" "fail_example" {
  rest_api_id   = "abc123"
  resource_id   = "res123"
  http_method   = "GET"
  authorization = "NONE"

  api_key_required = false
}
