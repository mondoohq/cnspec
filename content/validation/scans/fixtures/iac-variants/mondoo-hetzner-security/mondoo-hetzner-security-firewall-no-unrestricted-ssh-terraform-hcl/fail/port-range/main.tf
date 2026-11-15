resource "hcloud_firewall" "ssh" {
  name = "ssh"

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "20-30"
    source_ips = ["0.0.0.0/0"]
  }
}
