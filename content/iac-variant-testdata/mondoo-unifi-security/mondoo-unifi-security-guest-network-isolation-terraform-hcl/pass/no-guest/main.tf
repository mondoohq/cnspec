resource "unifi_network" "corp" {
  name    = "LAN"
  purpose = "corporate"
  subnet  = "192.168.1.0/24"
}

resource "unifi_network" "vlan" {
  name    = "vlan-only"
  purpose = "vlan-only"
  vlan_id = 40
}
