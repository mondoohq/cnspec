resource "google_vertex_ai_endpoint" "pass" {
  name         = "endpoint"
  display_name = "terraform-endpoint"
  location     = "us-central1"
  network      = "projects/123/global/networks/my-vpc"
}
