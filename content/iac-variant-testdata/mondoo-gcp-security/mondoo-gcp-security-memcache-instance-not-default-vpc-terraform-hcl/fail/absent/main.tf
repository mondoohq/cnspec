# Non-compliant: authorized_network is not set, so the instance falls back to the
# default VPC network.
resource "google_memcache_instance" "fail_example" {
  name   = "app-cache"
  region = "us-central1"

  node_config {
    cpu_count      = 1
    memory_size_mb = 1024
  }
  node_count = 1
}
