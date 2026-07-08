resource "hcloud_firewall" "db" {
  name = "db"

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "5432"
    source_ips = ["10.0.0.0/16"]
  }
}
