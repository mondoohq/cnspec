# Non-compliant: auth_enabled omitted (defaults to disabled).
resource "google_redis_instance" "cache" {
  name           = "default-cache"
  memory_size_gb = 1
  region         = "us-central1"
}
