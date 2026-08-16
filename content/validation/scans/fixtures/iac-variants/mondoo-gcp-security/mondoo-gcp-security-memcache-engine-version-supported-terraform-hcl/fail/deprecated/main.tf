# Non-compliant: memcache instance pins the deprecated MEMCACHE_1_5 engine.
resource "google_memcache_instance" "fail_example" {
  name             = "legacy-cache"
  region           = "us-central1"
  memcache_version = "MEMCACHE_1_5"

  node_config {
    cpu_count      = 1
    memory_size_mb = 1024
  }
  node_count = 1
}
