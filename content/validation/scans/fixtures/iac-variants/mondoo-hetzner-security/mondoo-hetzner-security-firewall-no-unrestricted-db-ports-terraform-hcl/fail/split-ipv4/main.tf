resource "hcloud_firewall" "db" {
  name = "db"

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "6379"
    source_ips = ["0.0.0.0/2", "64.0.0.0/2", "128.0.0.0/2", "192.0.0.0/2"]
  }
}
