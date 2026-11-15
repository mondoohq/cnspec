resource "hcloud_firewall" "app" {
  name = "app"

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "any"
    source_ips = ["::/0"]
  }
}
