resource "digitalocean_droplet" "win" {
  name   = "win-1"
  region = "nyc1"
  size   = "s-1vcpu-1gb"
  image  = "ubuntu-22-04-x64"
}

resource "digitalocean_firewall" "win" {
  name        = "win-fw"
  droplet_ids = [digitalocean_droplet.win.id]

  inbound_rule {
    protocol         = "tcp"
    port_range       = "3389"
    source_addresses = ["0.0.0.0/0"]
  }
}
