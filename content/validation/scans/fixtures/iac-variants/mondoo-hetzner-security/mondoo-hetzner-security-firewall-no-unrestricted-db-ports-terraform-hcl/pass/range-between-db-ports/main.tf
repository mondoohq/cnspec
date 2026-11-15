resource "hcloud_firewall" "app" {
  name = "app"

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "3307-5431"
    source_ips = ["0.0.0.0/0"]
  }
}
