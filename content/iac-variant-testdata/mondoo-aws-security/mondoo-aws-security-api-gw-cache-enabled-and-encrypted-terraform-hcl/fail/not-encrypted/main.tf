# Non-compliant: method settings do not encrypt cached data.
resource "aws_api_gateway_method_settings" "fail_example" {
  rest_api_id = "abc123"
  stage_name  = "prod"
  method_path = "*/*"

  settings {
    caching_enabled      = true
    cache_data_encrypted = false
  }
}
