resource "hcloud_firewall" "default" {
  name = "default"

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "22"
    source_ips = ["10.0.0.0/16"]
  }
}

resource "hcloud_server" "example" {
  name         = "example"
  image        = "ubuntu-24.04"
  server_type  = "cx22"
  location     = "nbg1"
  firewall_ids = [hcloud_firewall.default.id]
}
