# Compliant: AUTH is enabled on the Redis instance.
resource "google_redis_instance" "cache" {
  name           = "auth-cache"
  memory_size_gb = 1
  region         = "us-central1"
  auth_enabled   = true
}
