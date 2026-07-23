resource "google_vertex_ai_index_endpoint" "pass" {
  display_name            = "terraform-index-endpoint"
  region                  = "us-central1"
  network                 = "projects/123/global/networks/my-vpc"
  public_endpoint_enabled = false
}
