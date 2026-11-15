# A rule that trusts a tag rather than addresses admits no internet source.
resource "digitalocean_firewall" "web" {
  name = "web-fw"

  inbound_rule {
    protocol    = "tcp"
    port_range  = "22"
    source_tags = ["bastion"]
  }
}
