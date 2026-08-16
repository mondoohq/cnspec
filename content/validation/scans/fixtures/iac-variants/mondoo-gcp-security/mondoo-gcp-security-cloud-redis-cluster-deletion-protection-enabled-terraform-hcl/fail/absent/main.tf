# Non-compliant: deletion_protection_enabled omitted.
resource "google_redis_cluster" "prod" {
  name        = "prod-cluster"
  region      = "us-central1"
  shard_count = 3
}
