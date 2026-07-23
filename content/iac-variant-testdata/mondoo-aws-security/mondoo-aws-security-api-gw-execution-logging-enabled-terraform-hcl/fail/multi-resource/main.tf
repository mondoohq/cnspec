# Non-compliant: one of two method-settings resources disables execution logging.
resource "aws_api_gateway_method_settings" "ok" {
  rest_api_id = "abc123"
  stage_name  = "prod"
  method_path = "*/*"

  settings {
    logging_level = "INFO"
  }
}

resource "aws_api_gateway_method_settings" "bad" {
  rest_api_id = "abc123"
  stage_name  = "staging"
  method_path = "*/*"

  settings {
    logging_level = "OFF"
  }
}
