resource "hcloud_firewall" "rdp" {
  name = "rdp"

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "3000-4000"
    source_ips = ["0.0.0.0/0"]
  }
}
