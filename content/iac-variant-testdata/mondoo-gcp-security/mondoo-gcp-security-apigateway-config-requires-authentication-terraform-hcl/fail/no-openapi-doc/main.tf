# Non-compliant: config has no openapi_documents block.
resource "google_api_gateway_api_config" "fail_example" {
  api           = "my-api"
  api_config_id = "fail-config"
}
