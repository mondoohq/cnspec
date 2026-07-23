# Non-compliant: in-transit encryption disabled on the cluster.
resource "google_redis_cluster" "prod" {
  name                    = "plain-cluster"
  region                  = "us-central1"
  shard_count             = 3
  transit_encryption_mode = "TRANSIT_ENCRYPTION_MODE_DISABLED"
}
