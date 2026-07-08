# Compliant: Redis instance encrypted with a customer-managed key.
resource "google_redis_instance" "cache" {
  name                = "cmek-cache"
  memory_size_gb      = 1
  region              = "us-central1"
  customer_managed_key = "projects/my-project/locations/us-central1/keyRings/redis/cryptoKeys/cmek"
}
