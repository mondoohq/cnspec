resource "hcloud_firewall" "egress" {
  name = "egress"

  rule {
    direction       = "out"
    protocol        = "tcp"
    port            = "any"
    destination_ips = ["0.0.0.0/1", "128.0.0.0/1"]
  }
}
