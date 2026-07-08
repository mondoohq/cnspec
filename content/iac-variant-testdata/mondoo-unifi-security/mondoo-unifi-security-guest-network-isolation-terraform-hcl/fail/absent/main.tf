resource "unifi_network" "guest" {
  name    = "guest"
  purpose = "guest"
  subnet  = "192.168.30.1/24"
}
