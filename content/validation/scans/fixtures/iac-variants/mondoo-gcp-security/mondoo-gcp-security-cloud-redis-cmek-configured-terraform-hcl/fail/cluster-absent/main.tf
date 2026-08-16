# Non-compliant: cluster uses Google-managed encryption (no CMEK).
resource "google_redis_cluster" "prod" {
  name        = "gmek-cluster"
  region      = "us-central1"
  shard_count = 3
}
