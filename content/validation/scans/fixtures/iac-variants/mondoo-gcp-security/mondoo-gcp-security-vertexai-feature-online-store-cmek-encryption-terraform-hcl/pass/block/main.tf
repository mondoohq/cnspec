resource "google_vertex_ai_feature_online_store" "pass" {
  name   = "terraform_feature_online_store"
  region = "us-central1"

  bigtable {
    auto_scaling {
      min_node_count = 1
      max_node_count = 3
    }
  }

  encryption_spec {
    kms_key_name = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"
  }
}
