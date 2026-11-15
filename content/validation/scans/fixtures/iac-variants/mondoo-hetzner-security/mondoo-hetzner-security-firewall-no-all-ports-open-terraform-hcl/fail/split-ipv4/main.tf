resource "hcloud_firewall" "all" {
  name = "all"

  rule {
    direction  = "in"
    protocol   = "udp"
    port       = "1-65535"
    source_ips = ["0.0.0.0/1", "128.0.0.0/1"]
  }
}
