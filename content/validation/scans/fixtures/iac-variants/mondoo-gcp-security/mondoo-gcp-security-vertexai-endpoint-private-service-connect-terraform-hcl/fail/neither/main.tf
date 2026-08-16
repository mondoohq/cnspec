resource "google_vertex_ai_endpoint" "fail" {
  name         = "endpoint"
  display_name = "terraform-endpoint"
  location     = "us-central1"
}
