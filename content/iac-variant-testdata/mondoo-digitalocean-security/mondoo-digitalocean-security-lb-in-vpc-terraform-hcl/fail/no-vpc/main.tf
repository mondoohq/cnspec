resource "digitalocean_loadbalancer" "public" {
  name   = "public-lb"
  region = "nyc1"

  forwarding_rule {
    entry_port      = 443
    entry_protocol  = "https"
    target_port     = 80
    target_protocol = "http"
  }
}
