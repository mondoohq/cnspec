data "digitalocean_kubernetes_versions" "eos" {
  version_prefix = "1.33."
}

resource "digitalocean_kubernetes_cluster" "primary" {
  name    = "eos-cluster"
  region  = "nyc1"
  version = data.digitalocean_kubernetes_versions.eos.latest_version

  node_pool {
    name       = "worker-pool"
    size       = "s-2vcpu-2gb"
    node_count = 3
  }
}
