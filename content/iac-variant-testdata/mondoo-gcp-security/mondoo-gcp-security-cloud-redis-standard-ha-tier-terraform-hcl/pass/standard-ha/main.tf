# Compliant: Redis instance uses the STANDARD_HA tier.
resource "google_redis_instance" "cache" {
  name           = "ha-cache"
  memory_size_gb = 5
  region         = "us-central1"
  tier           = "STANDARD_HA"
}
