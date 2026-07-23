resource "unifi_wlan" "personal" {
  name          = "internal"
  security      = "wpapsk"
  passphrase    = var.wifi_passphrase
  network_id    = unifi_network.lan.id
  user_group_id = unifi_user_group.default.id
}
