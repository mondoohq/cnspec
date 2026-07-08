resource "google_vertex_ai_feature_online_store" "fail" {
  name   = "terraform_feature_online_store"
  region = "us-central1"

  bigtable {
    auto_scaling {
      min_node_count = 1
      max_node_count = 3
    }
  }
}
