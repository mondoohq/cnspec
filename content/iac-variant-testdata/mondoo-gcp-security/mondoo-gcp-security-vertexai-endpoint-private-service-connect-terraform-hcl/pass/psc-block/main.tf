resource "google_vertex_ai_endpoint" "pass" {
  name         = "endpoint"
  display_name = "terraform-endpoint"
  location     = "us-central1"

  private_service_connect_config {
    enable_private_service_connect = true
    project_allowlist              = ["my-project"]
  }
}
