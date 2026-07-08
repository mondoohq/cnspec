resource "hcloud_firewall" "internal" {
  name = "internal"

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "1-65535"
    source_ips = ["10.0.0.0/16"]
  }
}
