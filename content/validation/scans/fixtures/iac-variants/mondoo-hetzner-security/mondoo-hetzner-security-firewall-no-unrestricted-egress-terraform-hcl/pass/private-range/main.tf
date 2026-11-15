resource "hcloud_firewall" "egress" {
  name = "egress"

  rule {
    direction       = "out"
    protocol        = "tcp"
    port            = "any"
    destination_ips = ["10.0.0.0/8"]
  }
}
