# Compliant: memcache instance is attached to a dedicated custom VPC network.
resource "google_memcache_instance" "pass_example" {
  name               = "app-cache"
  region             = "us-central1"
  authorized_network = "projects/my-project/global/networks/app-vpc"

  node_config {
    cpu_count      = 1
    memory_size_mb = 1024
  }
  node_count = 1
}
