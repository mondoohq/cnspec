# Compliant: memcache instance runs a supported engine version.
resource "google_memcache_instance" "pass_example" {
  name            = "app-cache"
  region          = "us-central1"
  memcache_version = "MEMCACHE_1_6_15"

  node_config {
    cpu_count      = 1
    memory_size_mb = 1024
  }
  node_count = 1
}
