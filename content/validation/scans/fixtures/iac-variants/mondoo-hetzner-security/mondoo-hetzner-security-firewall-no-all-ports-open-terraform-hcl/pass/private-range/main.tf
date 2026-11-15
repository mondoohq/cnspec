resource "hcloud_firewall" "all" {
  name = "all"

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "any"
    source_ips = ["10.0.0.0/8"]
  }
}
