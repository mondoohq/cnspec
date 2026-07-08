# Non-compliant: deletion protection explicitly disabled.
resource "google_redis_cluster" "prod" {
  name                         = "prod-cluster"
  region                       = "us-central1"
  shard_count                  = 3
  deletion_protection_enabled  = false
}
