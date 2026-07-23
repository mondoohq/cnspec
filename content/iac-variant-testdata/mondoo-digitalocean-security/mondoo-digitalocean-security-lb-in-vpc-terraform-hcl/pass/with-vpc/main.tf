resource "digitalocean_vpc" "app" {
  name   = "app-vpc"
  region = "nyc1"
}

resource "digitalocean_loadbalancer" "public" {
  name     = "public-lb"
  region   = "nyc1"
  vpc_uuid = digitalocean_vpc.app.id

  forwarding_rule {
    entry_port      = 443
    entry_protocol  = "https"
    target_port     = 80
    target_protocol = "http"
  }
}
