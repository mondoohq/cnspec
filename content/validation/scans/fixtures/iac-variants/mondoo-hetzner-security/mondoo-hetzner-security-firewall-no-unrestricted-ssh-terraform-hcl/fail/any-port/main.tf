resource "hcloud_firewall" "ssh" {
  name = "ssh"

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "any"
    source_ips = ["::/0"]
  }
}
