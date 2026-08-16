data "digitalocean_kubernetes_versions" "supported" {
  version_prefix = "1.31."
}

resource "digitalocean_kubernetes_cluster" "primary" {
  name    = "prod-cluster"
  region  = "nyc1"
  version = data.digitalocean_kubernetes_versions.supported.latest_version

  node_pool {
    name       = "worker-pool"
    size       = "s-2vcpu-2gb"
    node_count = 3
  }
}
