resource "google_vertex_ai_featurestore" "pass" {
  name   = "terraform_featurestore"
  region = "us-central1"

  online_serving_config {
    fixed_node_count = 2
  }

  encryption_spec {
    kms_key_name = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"
  }
}
