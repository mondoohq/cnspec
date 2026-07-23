# Compliant: in-transit encryption enabled on the Redis instance.
resource "google_redis_instance" "cache" {
  name                    = "tls-cache"
  memory_size_gb          = 1
  region                  = "us-central1"
  transit_encryption_mode = "SERVER_AUTHENTICATION"
}
