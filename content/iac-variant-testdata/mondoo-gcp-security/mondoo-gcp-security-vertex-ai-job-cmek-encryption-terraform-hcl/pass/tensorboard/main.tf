resource "google_vertex_ai_tensorboard" "pass" {
  display_name = "terraform-tensorboard"
  region       = "us-central1"

  encryption_spec {
    kms_key_name = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"
  }
}
