# Non-compliant: tier omitted (defaults to BASIC).
resource "google_redis_instance" "cache" {
  name           = "default-cache"
  memory_size_gb = 1
  region         = "us-central1"
}
