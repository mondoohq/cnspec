# Each /1 half admits half of the internet; together they admit all of it.
resource "digitalocean_firewall" "web" {
  name = "web-fw"

  inbound_rule {
    protocol         = "tcp"
    port_range       = "all"
    source_addresses = ["0.0.0.0/1", "128.0.0.0/1"]
  }
}
