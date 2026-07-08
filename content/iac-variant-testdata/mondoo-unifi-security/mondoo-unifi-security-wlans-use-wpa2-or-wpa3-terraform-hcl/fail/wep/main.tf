resource "unifi_wlan" "legacy" {
  name          = "legacy"
  security      = "wep"
  network_id    = unifi_network.lan.id
  user_group_id = unifi_user_group.default.id
}
