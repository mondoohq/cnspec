# Non-compliant: AUTH explicitly disabled.
resource "google_redis_instance" "cache" {
  name           = "noauth-cache"
  memory_size_gb = 1
  region         = "us-central1"
  auth_enabled   = false
}
