resource "google_vertex_ai_metadata_store" "fail" {
  name        = "terraform-metadata-store"
  description = "store without CMEK"
  region      = "us-central1"
}
