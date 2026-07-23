# Non-compliant: a counted method-settings resource leaves cache unencrypted.
resource "aws_api_gateway_method_settings" "fail_count" {
  count       = 2
  rest_api_id = "abc123"
  stage_name  = "prod-${count.index}"
  method_path = "*/*"

  settings {
    caching_enabled      = true
    cache_data_encrypted = false
  }
}
