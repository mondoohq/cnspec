# Non-compliant: Redis instance uses the BASIC (non-HA) tier.
resource "google_redis_instance" "cache" {
  name           = "basic-cache"
  memory_size_gb = 1
  region         = "us-central1"
  tier           = "BASIC"
}
