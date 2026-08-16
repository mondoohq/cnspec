data "digitalocean_kubernetes_versions" "legacy" {
  version_prefix = "1.29."
}

resource "digitalocean_kubernetes_cluster" "primary" {
  name    = "legacy-cluster"
  region  = "nyc1"
  version = data.digitalocean_kubernetes_versions.legacy.latest_version

  node_pool {
    name       = "worker-pool"
    size       = "s-2vcpu-2gb"
    node_count = 3
  }
}
