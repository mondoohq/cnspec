# Compliant: memcache_version is not set, so it defaults to the current supported
# version rather than the deprecated MEMCACHE_1_5.
resource "google_memcache_instance" "pass_example" {
  name   = "app-cache"
  region = "us-central1"

  node_config {
    cpu_count      = 1
    memory_size_mb = 1024
  }
  node_count = 1
}
