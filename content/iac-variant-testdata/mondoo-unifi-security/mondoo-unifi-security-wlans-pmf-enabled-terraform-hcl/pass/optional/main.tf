resource "unifi_wlan" "secure" {
  name          = "corp"
  security      = "wpaeap"
  network_id    = unifi_network.lan.id
  user_group_id = unifi_user_group.default.id
  pmf_mode      = "optional"
}
