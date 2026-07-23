resource "digitalocean_droplet" "db" {
  name   = "db-1"
  region = "nyc1"
  size   = "s-1vcpu-1gb"
  image  = "ubuntu-22-04-x64"
}

resource "digitalocean_firewall" "db" {
  name        = "db-fw"
  droplet_ids = [digitalocean_droplet.db.id]

  inbound_rule {
    protocol         = "tcp"
    port_range       = "5432"
    source_addresses = ["10.0.0.0/8"]
  }
}
