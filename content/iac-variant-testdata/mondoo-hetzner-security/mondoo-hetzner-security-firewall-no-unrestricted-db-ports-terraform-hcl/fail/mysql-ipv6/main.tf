resource "hcloud_firewall" "db" {
  name = "db"

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "3306"
    source_ips = ["::/0"]
  }
}
