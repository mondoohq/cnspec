resource "hcloud_firewall" "ssh" {
  name = "ssh"

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "22"
    source_ips = ["10.0.0.0/8", "fc00::/7"]
  }
}
