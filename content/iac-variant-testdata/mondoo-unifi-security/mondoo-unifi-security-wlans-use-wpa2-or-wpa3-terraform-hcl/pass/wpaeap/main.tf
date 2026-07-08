resource "unifi_wlan" "enterprise" {
  name          = "corp-8021x"
  security      = "wpaeap"
  network_id    = unifi_network.lan.id
  user_group_id = unifi_user_group.default.id
}
