resource "google_vertex_ai_index_endpoint" "pass" {
  display_name = "terraform-index-endpoint"
  region       = "us-central1"
  network      = "projects/123/global/networks/my-vpc"

  encryption_spec {
    kms_key_name = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"
  }
}
