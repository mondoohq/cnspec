resource "hcloud_firewall" "app" {
  name = "app"

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "5000-6000"
    source_ips = ["0.0.0.0/0"]
  }
}
