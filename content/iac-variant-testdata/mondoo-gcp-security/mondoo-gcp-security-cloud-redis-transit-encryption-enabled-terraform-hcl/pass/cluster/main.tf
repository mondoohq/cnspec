# Compliant: in-transit encryption enabled on the Redis cluster.
resource "google_redis_cluster" "prod" {
  name                    = "tls-cluster"
  region                  = "us-central1"
  shard_count             = 3
  transit_encryption_mode = "TRANSIT_ENCRYPTION_MODE_SERVER_AUTHENTICATION"
}
