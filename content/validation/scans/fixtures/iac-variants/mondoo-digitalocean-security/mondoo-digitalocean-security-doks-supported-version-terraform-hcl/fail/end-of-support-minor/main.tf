resource "digitalocean_kubernetes_cluster" "primary" {
  name    = "eos-cluster"
  region  = "nyc1"
  version = "1.33.1-do.0"

  node_pool {
    name       = "worker-pool"
    size       = "s-2vcpu-2gb"
    node_count = 3
  }
}
