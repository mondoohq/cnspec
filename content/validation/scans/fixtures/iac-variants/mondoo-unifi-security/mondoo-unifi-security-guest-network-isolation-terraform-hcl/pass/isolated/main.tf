resource "unifi_network" "corp" {
  name    = "LAN"
  purpose = "corporate"
  subnet  = "192.168.1.0/24"
}

resource "unifi_network" "guest" {
  name              = "guest"
  purpose           = "guest"
  subnet            = "192.168.30.1/24"
  network_isolation = true
}
