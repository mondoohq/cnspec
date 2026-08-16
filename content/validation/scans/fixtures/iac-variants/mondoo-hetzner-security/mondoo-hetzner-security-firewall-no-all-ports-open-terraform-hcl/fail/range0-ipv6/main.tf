resource "hcloud_firewall" "open" {
  name = "open"

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "0-65535"
    source_ips = ["::/0"]
  }
}
