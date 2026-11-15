resource "hcloud_firewall" "web" {
  name = "web"

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "23-1024"
    source_ips = ["0.0.0.0/0"]
  }
}
