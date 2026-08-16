resource "hcloud_firewall" "egress" {
  name = "egress"

  rule {
    direction       = "out"
    protocol        = "tcp"
    port            = "0-65535"
    destination_ips = ["::/0"]
  }
}
