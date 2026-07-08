resource "digitalocean_kubernetes_cluster" "primary" {
  name    = "primary"
  region  = "nyc1"
  version = "1.30.1-do.0"

  node_pool {
    name       = "default"
    size       = "s-1vcpu-2gb"
    node_count = 2
  }

  control_plane_firewall {
    enabled           = false
    allowed_addresses = ["203.0.113.0/24"]
  }
}
