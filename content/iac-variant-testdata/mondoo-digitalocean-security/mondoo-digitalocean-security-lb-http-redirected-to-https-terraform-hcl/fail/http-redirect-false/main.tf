resource "digitalocean_loadbalancer" "public" {
  name                   = "public-lb"
  region                 = "nyc1"
  vpc_uuid               = "0d3176ad-41e0-4021-b831-0c5c45c60959"
  redirect_http_to_https = false

  forwarding_rule {
    entry_port      = 80
    entry_protocol  = "http"
    target_port     = 80
    target_protocol = "http"
  }
}
