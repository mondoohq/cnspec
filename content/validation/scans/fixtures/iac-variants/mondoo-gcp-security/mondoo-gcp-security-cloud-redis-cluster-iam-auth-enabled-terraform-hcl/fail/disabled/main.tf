# Non-compliant: authentication disabled.
resource "google_redis_cluster" "prod" {
  name               = "prod-cluster"
  region             = "us-central1"
  shard_count        = 3
  authorization_mode = "AUTH_MODE_DISABLED"
}
