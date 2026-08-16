# Compliant: method settings enable cache data encryption.
resource "aws_api_gateway_method_settings" "pass_example" {
  rest_api_id = "abc123"
  stage_name  = "prod"
  method_path = "*/*"

  settings {
    caching_enabled      = true
    cache_data_encrypted = true
  }
}
