# Non-compliant: CMEK attribute present but empty.
resource "google_redis_instance" "cache" {
  name                 = "empty-cmek-cache"
  memory_size_gb       = 1
  region               = "us-central1"
  customer_managed_key = ""
}
