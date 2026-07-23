# Non-compliant: in-transit encryption disabled on the instance.
resource "google_redis_instance" "cache" {
  name                    = "plain-cache"
  memory_size_gb          = 1
  region                  = "us-central1"
  transit_encryption_mode = "DISABLED"
}
