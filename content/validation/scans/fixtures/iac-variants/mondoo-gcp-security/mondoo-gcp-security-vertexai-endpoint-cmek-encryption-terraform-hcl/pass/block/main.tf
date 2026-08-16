resource "google_vertex_ai_endpoint" "pass" {
  name         = "endpoint"
  display_name = "terraform-endpoint"
  location     = "us-central1"

  encryption_spec {
    kms_key_name = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"
  }
}
