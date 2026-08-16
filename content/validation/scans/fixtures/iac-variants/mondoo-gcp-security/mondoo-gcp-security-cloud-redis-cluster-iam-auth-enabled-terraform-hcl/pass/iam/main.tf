# Compliant: cluster uses IAM authentication.
resource "google_redis_cluster" "prod" {
  name               = "prod-cluster"
  region             = "us-central1"
  shard_count        = 3
  authorization_mode = "AUTH_MODE_IAM_AUTH"
}
