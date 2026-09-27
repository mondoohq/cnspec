# A minor release with no entry in the lifecycle table fails rather than passing unexamined.
resource "digitalocean_kubernetes_cluster" "primary" {
  name    = "future-cluster"
  region  = "nyc1"
  version = "1.99.0-do.0"

  node_pool {
    name       = "worker-pool"
    size       = "s-2vcpu-2gb"
    node_count = 3
  }
}
