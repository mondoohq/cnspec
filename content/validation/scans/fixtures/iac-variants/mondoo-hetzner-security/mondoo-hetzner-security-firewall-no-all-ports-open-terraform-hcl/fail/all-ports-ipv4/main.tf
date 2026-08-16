resource "hcloud_firewall" "open" {
  name = "open"

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "1-65535"
    source_ips = ["0.0.0.0/0"]
  }
}
