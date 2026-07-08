resource "unifi_port_forward" "admin" {
  name    = "admin"
  wan     = { port = "443" }
  forward = { ip = "192.168.1.100", port = "8000" }
}
