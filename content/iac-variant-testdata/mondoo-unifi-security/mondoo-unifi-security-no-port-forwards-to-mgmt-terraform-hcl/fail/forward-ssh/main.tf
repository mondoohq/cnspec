resource "unifi_port_forward" "ssh" {
  name    = "ssh"
  wan     = { port = "2222" }
  forward = { ip = "192.168.1.10", port = "22" }
}
