# A non-canonical spelling of ::/0.
resource "digitalocean_firewall" "web" {
  name = "web-fw"

  inbound_rule {
    protocol         = "tcp"
    port_range       = "all"
    source_addresses = ["0::/0"]
  }
}
