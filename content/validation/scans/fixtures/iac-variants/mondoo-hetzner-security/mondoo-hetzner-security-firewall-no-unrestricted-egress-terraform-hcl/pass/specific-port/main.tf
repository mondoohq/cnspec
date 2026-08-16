resource "hcloud_firewall" "egress" {
  name = "egress"

  rule {
    direction       = "out"
    protocol        = "tcp"
    port            = "443"
    destination_ips = ["0.0.0.0/0"]
  }
}
