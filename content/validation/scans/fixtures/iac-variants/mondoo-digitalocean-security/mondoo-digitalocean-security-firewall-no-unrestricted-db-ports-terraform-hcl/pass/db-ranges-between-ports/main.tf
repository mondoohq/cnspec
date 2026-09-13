resource "digitalocean_droplet" "web" {
  name   = "web-1"
  region = "nyc1"
  size   = "s-1vcpu-1gb"
  image  = "ubuntu-22-04-x64"
}

resource "digitalocean_firewall" "web" {
  name        = "web-fw"
  droplet_ids = [digitalocean_droplet.web.id]

  inbound_rule {
    protocol         = "tcp"
    port_range       = "1-3305"
    source_addresses = ["0.0.0.0/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "3307-5431"
    source_addresses = ["0.0.0.0/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "5433-6378"
    source_addresses = ["0.0.0.0/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "6380-9091"
    source_addresses = ["0.0.0.0/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "9093-9199"
    source_addresses = ["0.0.0.0/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "9201-27016"
    source_addresses = ["0.0.0.0/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "27018-30000"
    source_addresses = ["::/0"]
  }
}
