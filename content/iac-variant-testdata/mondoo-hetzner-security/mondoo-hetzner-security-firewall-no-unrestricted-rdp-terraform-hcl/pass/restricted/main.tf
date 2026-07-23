resource "hcloud_firewall" "rdp" {
  name = "rdp"

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "3389"
    source_ips = ["203.0.113.0/24"]
  }
}
