resource "digitalocean_loadbalancer" "public" {
  name     = "public-lb"
  region   = "nyc1"
  vpc_uuid = "0d3176ad-41e0-4021-b831-0c5c45c60959"

  forwarding_rule {
    entry_port       = 443
    entry_protocol   = "https"
    target_port      = 80
    target_protocol  = "http"
    certificate_name = "cert-example"
  }
}
