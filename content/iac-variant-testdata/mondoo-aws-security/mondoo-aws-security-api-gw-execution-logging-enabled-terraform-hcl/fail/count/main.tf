# Non-compliant: a counted method-settings resource disables execution logging.
resource "aws_api_gateway_method_settings" "fail_count" {
  count       = 2
  rest_api_id = "abc123"
  stage_name  = "prod-${count.index}"
  method_path = "*/*"

  settings {
    logging_level = "OFF"
  }
}
