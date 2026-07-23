# Non-compliant: authorization_mode omitted (defaults to disabled).
resource "google_redis_cluster" "prod" {
  name        = "prod-cluster"
  region      = "us-central1"
  shard_count = 3
}
