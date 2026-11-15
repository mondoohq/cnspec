resource "hcloud_firewall" "web" {
  name = "web"

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "3390-4000"
    source_ips = ["0.0.0.0/0"]
  }
}
