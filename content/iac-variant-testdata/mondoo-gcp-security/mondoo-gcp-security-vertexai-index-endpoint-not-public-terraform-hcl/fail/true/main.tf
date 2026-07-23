resource "google_vertex_ai_index_endpoint" "fail" {
  display_name            = "terraform-index-endpoint"
  region                  = "us-central1"
  public_endpoint_enabled = true
}
