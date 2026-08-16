# Non-compliant: one of two method-settings resources leaves cache unencrypted.
resource "aws_api_gateway_method_settings" "ok" {
  rest_api_id = "abc123"
  stage_name  = "prod"
  method_path = "*/*"

  settings {
    caching_enabled      = true
    cache_data_encrypted = true
  }
}

resource "aws_api_gateway_method_settings" "bad" {
  rest_api_id = "abc123"
  stage_name  = "staging"
  method_path = "*/*"

  settings {
    caching_enabled      = true
    cache_data_encrypted = false
  }
}
