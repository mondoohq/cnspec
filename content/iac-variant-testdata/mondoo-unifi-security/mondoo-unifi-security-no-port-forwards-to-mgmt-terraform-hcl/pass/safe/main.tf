resource "unifi_port_forward" "web" {
  name    = "web"
  wan     = { port = "80" }
  forward = { ip = "192.168.1.100", port = "8000" }
}
