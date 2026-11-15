# All global unicast IPv6 space, written uncompressed.
resource "digitalocean_firewall" "web" {
  name = "web-fw"

  inbound_rule {
    protocol         = "tcp"
    port_range       = "22"
    source_addresses = ["2000:0::/3"]
  }
}
