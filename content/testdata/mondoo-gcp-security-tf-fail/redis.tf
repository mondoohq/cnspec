# Cloud Memorystore Redis fail fixture - every Redis check should fail.
#
# - auth_enabled defaults to false.
# - transit_encryption_mode unset (DISABLED by default).
# - No customer_managed_key.
# - tier defaults to BASIC.

resource "google_redis_instance" "cache" {
  name           = "fail-cache-${random_id.suffix.hex}"
  memory_size_gb = 1
  region         = "us-central1"
  # tier defaults to BASIC
  # auth_enabled defaults to false
  # transit_encryption_mode defaults to DISABLED
  # customer_managed_key intentionally absent
}
