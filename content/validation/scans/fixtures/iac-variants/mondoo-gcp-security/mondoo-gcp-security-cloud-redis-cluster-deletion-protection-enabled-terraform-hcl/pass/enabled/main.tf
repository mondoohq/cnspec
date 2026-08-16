# Compliant: deletion protection enabled on the Redis cluster.
resource "google_redis_cluster" "prod" {
  name                         = "prod-cluster"
  region                       = "us-central1"
  shard_count                  = 3
  deletion_protection_enabled  = true
}
