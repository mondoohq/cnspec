resource "hcloud_firewall" "ssh" {
  name = "ssh"

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "22"
    source_ips = ["::/0"]
  }
}
