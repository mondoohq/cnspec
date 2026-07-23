resource "unifi_network" "lan" {
  name          = "LAN"
  purpose       = "corporate"
  subnet        = "192.168.1.0/24"
  igmp_snooping = true
}
