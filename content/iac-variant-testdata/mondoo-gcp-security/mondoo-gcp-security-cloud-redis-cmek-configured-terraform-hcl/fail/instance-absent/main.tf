# Non-compliant: instance uses Google-managed encryption (no CMEK).
resource "google_redis_instance" "cache" {
  name           = "gmek-cache"
  memory_size_gb = 1
  region         = "us-central1"
}
