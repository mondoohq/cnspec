# Compliant: config declares an openapi_documents block.
resource "google_api_gateway_api_config" "pass_example" {
  api           = "my-api"
  api_config_id = "pass-config"

  openapi_documents {
    document {
      path     = "spec.yaml"
      contents = "b3BlbmFwaTogMy4wLjA="
    }
  }
}
