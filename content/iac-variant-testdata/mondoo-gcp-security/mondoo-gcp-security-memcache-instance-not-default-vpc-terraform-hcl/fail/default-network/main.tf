# Non-compliant: memcache instance is attached to the default VPC network.
resource "google_memcache_instance" "fail_example" {
  name               = "app-cache"
  region             = "us-central1"
  authorized_network = "projects/my-project/global/networks/default"

  node_config {
    cpu_count      = 1
    memory_size_mb = 1024
  }
  node_count = 1
}
